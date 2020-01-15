package process

import (
	"fmt"
	"github.com/ochinchina/filechangemonitor"
	"github.com/ochinchina/supervisord/config"
	"github.com/ochinchina/supervisord/events"
	"github.com/ochinchina/supervisord/logger"
	"github.com/ochinchina/supervisord/signals"
	"github.com/robfig/cron/v3"
	log "github.com/sirupsen/logrus"
	"io"
	"os"
	"os/exec"
	"os/user"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

type ProcessState int

const (
	STOPPED  ProcessState = iota
	STARTING              = 10
	RUNNING               = 20
	BACKOFF               = 30
	STOPPING              = 40
	EXITED                = 100
	FATAL                 = 200
	UNKNOWN               = 1000
)

var scheduler *cron.Cron = nil

func init() {
	scheduler = cron.New(cron.WithSeconds())
	scheduler.Start()
}

func (p ProcessState) String() string {
	switch p {
	case STOPPED:
		return "STOPPED"
	case STARTING:
		return "STARTING"
	case RUNNING:
		return "RUNNING"
	case BACKOFF:
		return "BACKOFF"
	case STOPPING:
		return "STOPPING"
	case EXITED:
		return "EXITED"
	case FATAL:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

type Process struct {
	supervisor_id string
	config        *config.ConfigEntry
	cmd           *exec.Cmd
	startTime     time.Time
	stopTime      time.Time
	state         ProcessState
	//true if process is starting
	inStart bool
	//true if the process is stopped by user
	stopByUser bool
	retryTimes *int32
	lock       sync.RWMutex
	stdin      io.WriteCloser
	StdoutLog  logger.Logger
	StderrLog  logger.Logger
}

func NewProcess(supervisor_id string, config *config.ConfigEntry) *Process {
	proc := &Process{supervisor_id: supervisor_id,
		config:     config,
		cmd:        nil,
		startTime:  time.Unix(0, 0),
		stopTime:   time.Unix(0, 0),
		state:      STOPPED,
		inStart:    false,
		stopByUser: false,
		retryTimes: new(int32)}
	proc.config = config
	proc.cmd = nil
	proc.addToCron()
	return proc
}

// add this process to crontab
func (p *Process) addToCron() {
	s := p.config.GetString("cron", "")

	if s != "" {
		log.WithFields(log.Fields{"program": p.GetName()}).Info("try to create cron program with cron expression:", s)
		scheduler.AddFunc(s, func() {
			log.WithFields(log.Fields{"program": p.GetName()}).Info("start cron program")
			if !p.isRunning() {
				p.Start(false)
			}
		})
	}

}

// start the process
// Args:
//  wait - true, wait the program started or failed
func (p *Process) Start(wait bool) {
	log.WithFields(log.Fields{"program": p.GetName()}).Info("try to start program")
	p.lock.Lock()
	if p.inStart {
		log.WithFields(log.Fields{"program": p.GetName()}).Info("Don't start program again, program is already started")
		p.lock.Unlock()
		return
	}

	p.inStart = true
	p.stopByUser = false
	p.lock.Unlock()

	var runCond *sync.Cond
	finished := false
	if wait {
		runCond = sync.NewCond(&sync.Mutex{})
		runCond.L.Lock()
	}

	go func() {

		for {
			if wait {
				runCond.L.Lock()
			}
			p.run(func() {
				finished = true
				if wait {
					runCond.L.Unlock()
					runCond.Signal()
				}
			})
			//avoid print too many logs if fail to start program too quickly
			if time.Now().Unix()-p.startTime.Unix() < 2 {
				time.Sleep(5 * time.Second)
			}
			if p.stopByUser {
				log.WithFields(log.Fields{"program": p.GetName()}).Info("Stopped by user, don't start it again")
				break
			}
			if !p.isAutoRestart() {
				log.WithFields(log.Fields{"program": p.GetName()}).Info("Don't start the stopped program because its autorestart flag is false")
				break
			}
		}
		p.lock.Lock()
		p.inStart = false
		p.lock.Unlock()
	}()
	if wait && !finished {
		runCond.Wait()
		runCond.L.Unlock()
	}
}

func (p *Process) GetName() string {
	if p.config.IsProgram() {
		return p.config.GetProgramName()
	} else if p.config.IsEventListener() {
		return p.config.GetEventListenerName()
	} else {
		return ""
	}
}

func (p *Process) GetGroup() string {
	return p.config.Group
}

func (p *Process) GetDescription() string {
	p.lock.RLock()
	defer p.lock.RUnlock()
	if p.state == RUNNING {
		seconds := int(time.Now().Sub(p.startTime).Seconds())
		minutes := seconds / 60
		hours := minutes / 60
		days := hours / 24
		if days > 0 {
			return fmt.Sprintf("pid %d, uptime %d days, %d:%02d:%02d", p.cmd.Process.Pid, days, hours%24, minutes%60, seconds%60)
		}
		return fmt.Sprintf("pid %d, uptime %d:%02d:%02d", p.cmd.Process.Pid, hours%24, minutes%60, seconds%60)
	} else if p.state != STOPPED {
		return p.stopTime.String()
	}
	return ""
}

func (p *Process) GetExitstatus() int {
	p.lock.RLock()
	defer p.lock.RUnlock()

	if p.state == EXITED || p.state == BACKOFF {
		if p.cmd.ProcessState == nil {
			return 0
		}
		status, ok := p.cmd.ProcessState.Sys().(syscall.WaitStatus)
		if ok {
			return status.ExitStatus()
		}
	}
	return 0
}

func (p *Process) GetPid() int {
	p.lock.RLock()
	defer p.lock.RUnlock()

	if p.state == STOPPED || p.state == FATAL || p.state == UNKNOWN || p.state == EXITED || p.state == BACKOFF {
		return 0
	}
	return p.cmd.Process.Pid
}

// Get the process state
func (p *Process) GetState() ProcessState {
	return p.state
}

func (p *Process) GetStartTime() time.Time {
	return p.startTime
}

func (p *Process) GetStopTime() time.Time {
	switch p.state {
	case STARTING:
		fallthrough
	case RUNNING:
		fallthrough
	case STOPPING:
		return time.Unix(0, 0)
	default:
		return p.stopTime
	}
}

func (p *Process) GetStdoutLogfile() string {
	file_name := p.config.GetStringExpression("stdout_logfile", "/dev/null")
	expand_file, err := Path_expand(file_name)
	if err != nil {
		return file_name
	}
	return expand_file
}

func (p *Process) GetStderrLogfile() string {
	file_name := p.config.GetStringExpression("stderr_logfile", "/dev/null")
	expand_file, err := Path_expand(file_name)
	if err != nil {
		return file_name
	}
	return expand_file
}

func (p *Process) getStartSeconds() int64 {
	return int64(p.config.GetInt("startsecs", 1))
}

func (p *Process) getRestartPause() int {
	return p.config.GetInt("restartpause", 0)
}

func (p *Process) getStartRetries() int32 {
	return int32(p.config.GetInt("startretries", 3))
}

func (p *Process) isAutoStart() bool {
	return p.config.GetString("autostart", "true") == "true"
}

func (p *Process) GetPriority() int {
	return p.config.GetInt("priority", 999)
}

func (p *Process) getNumberProcs() int {
	return p.config.GetInt("numprocs", 1)
}

func (p *Process) SendProcessStdin(chars string) error {
	if p.stdin != nil {
		_, err := p.stdin.Write([]byte(chars))
		return err
	}
	return fmt.Errorf("NO_FILE")
}

// check if the process should be
func (p *Process) isAutoRestart() bool {
	autoRestart := p.config.GetString("autorestart", "unexpected")

	if autoRestart == "false" {
		return false
	} else if autoRestart == "true" {
		return true
	} else {
		p.lock.RLock()
		defer p.lock.RUnlock()
		if p.cmd != nil && p.cmd.ProcessState != nil {
			exitCode, err := p.getExitCode()
			//If unexpected, the process will be restarted when the program exits
			//with an exit code that is not one of the exit codes associated with
			//this process’ configuration (see exitcodes).
			return err == nil && !p.inExitCodes(exitCode)
		}
	}
	return false

}

func (p *Process) inExitCodes(exitCode int) bool {
	for _, code := range p.getExitCodes() {
		if code == exitCode {
			return true
		}
	}
	return false
}

func (p *Process) getExitCode() (int, error) {
	if p.cmd.ProcessState == nil {
		return -1, fmt.Errorf("no exit code")
	}
	if status, ok := p.cmd.ProcessState.Sys().(syscall.WaitStatus); ok {
		return status.ExitStatus(), nil
	}

	return -1, fmt.Errorf("no exit code")

}

func (p *Process) getExitCodes() []int {
	strExitCodes := strings.Split(p.config.GetString("exitcodes", "0,2"), ",")
	result := make([]int, 0)
	for _, val := range strExitCodes {
		i, err := strconv.Atoi(val)
		if err == nil {
			result = append(result, i)
		}
	}
	return result
}

// check if the process is running or not
//
func (p *Process) isRunning() bool {
	if p.cmd != nil && p.cmd.Process != nil {
		if runtime.GOOS == "windows" {
			proc, err := os.FindProcess(p.cmd.Process.Pid)
			return proc != nil && err == nil
		} else {
			fmt.Printf("send signal 0 to process\n")
			return p.cmd.Process.Signal(syscall.Signal(0)) == nil
		}
	}
	return false
}

// create Command object for the program
func (p *Process) createProgramCommand() error {
	args, err := parseCommand(p.config.GetStringExpression("command", ""))

	if err != nil {
		return err
	}
	p.cmd = exec.Command(args[0])
	if len(args) > 1 {
		p.cmd.Args = args
	}
	p.cmd.SysProcAttr = &syscall.SysProcAttr{}
	if p.setUser() != nil {
		log.WithFields(log.Fields{"user": p.config.GetString("user", "")}).Error("fail to run as user")
		return fmt.Errorf("fail to set user")
	}
	p.setProgramRestartChangeMonitor(args[0])
	set_deathsig(p.cmd.SysProcAttr)
	p.setEnv()
	p.setDir()
	p.setLog()

	p.stdin, _ = p.cmd.StdinPipe()
	return nil

}

func (p *Process) setProgramRestartChangeMonitor(programPath string) {
	if p.config.GetBool("restart_when_binary_changed", false) {
		AddProgramChangeMonitor(programPath, func(path string, mode filechangemonitor.FileChangeMode) {
			log.WithFields(log.Fields{"program": p.GetName()}).Info("program is changed, resatrt it")
			p.Stop(true)
			p.Start(true)
		})
	}
	dir_monitor := p.config.GetString("restart_directory_monitor", "")
	file_pattern := p.config.GetString("restart_file_pattern", "*")
	if dir_monitor != "" {
		AddConfigChangeMonitor(dir_monitor, file_pattern, func(path string, mode filechangemonitor.FileChangeMode) {
			//fmt.Printf( "file_pattern=%s, base=%s\n", file_pattern, filepath.Base( path ) )
			//if matched, err := filepath.Match( file_pattern, filepath.Base( path ) ); matched && err == nil {
			log.WithFields(log.Fields{"program": p.GetName()}).Info("configure file for program is changed, resatrt it")
			p.Stop(true)
			p.Start(true)
			//}
		})
	}

}

// wait for the started program exit
func (p *Process) waitForExit(startSecs int64) {
	err := p.cmd.Wait()
	if err != nil {
		log.WithFields(log.Fields{"program": p.GetName()}).Info("fail to wait for program exit")
	} else if p.cmd.ProcessState != nil {
		log.WithFields(log.Fields{"program": p.GetName()}).Infof("program stopped with status:%v", p.cmd.ProcessState)
	} else {
		log.WithFields(log.Fields{"program": p.GetName()}).Info("program stopped")
	}
	p.lock.Lock()
	defer p.lock.Unlock()
	p.stopTime = time.Now()
	p.StdoutLog.Close()
	p.StderrLog.Close()
}

// fail to start the program
func (p *Process) failToStartProgram(reason string, finishCb func()) {
	log.WithFields(log.Fields{"program": p.GetName()}).Errorf(reason)
	p.changeStateTo(FATAL)
	finishCb()
}

// monitor if the program is in running before endTime
//
func (p *Process) monitorProgramIsRunning(endTime time.Time, monitorExited *int32, programExited *int32) {
	// if time is not expired
	for time.Now().Before(endTime) && atomic.LoadInt32(programExited) == 0 {
		time.Sleep(time.Duration(100) * time.Millisecond)
	}
	atomic.StoreInt32(monitorExited, 1)

	p.lock.Lock()
	defer p.lock.Unlock()
	// if the program does not exit
	if atomic.LoadInt32(programExited) == 0 && p.state == STARTING {
		log.WithFields(log.Fields{"program": p.GetName()}).Info("success to start program")
		p.changeStateTo(RUNNING)
	}
}

func (p *Process) run(finishCb func()) {
	p.lock.Lock()
	defer p.lock.Unlock()

	// check if the program is in running state
	if p.isRunning() {
		log.WithFields(log.Fields{"program": p.GetName()}).Info("Don't start program because it is running")
		finishCb()
		return

	}
	p.startTime = time.Now()
	atomic.StoreInt32(p.retryTimes, 0)
	startSecs := p.getStartSeconds()
	restartPause := p.getRestartPause()
	var once sync.Once

	// finishCb can be only called one time
	finishCbWrapper := func() {
		once.Do(finishCb)
	}
	//process is not expired and not stoped by user
	for !p.stopByUser {
		if restartPause > 0 && atomic.LoadInt32(p.retryTimes) != 0 {
			//pause
			p.lock.Unlock()
			log.WithFields(log.Fields{"program": p.GetName()}).Info("don't restart the program, start it after ", restartPause, " seconds")
			time.Sleep(time.Duration(restartPause) * time.Second)
			p.lock.Lock()
		}
		endTime := time.Now().Add(time.Duration(startSecs) * time.Second)
		p.changeStateTo(STARTING)
		atomic.AddInt32(p.retryTimes, 1)

		err := p.createProgramCommand()
		if err != nil {
			p.failToStartProgram("fail to create program", finishCbWrapper)
			break
		}

		err = p.cmd.Start()

		if err != nil {
			if atomic.LoadInt32(p.retryTimes) >= p.getStartRetries() {
				p.failToStartProgram(fmt.Sprintf("fail to start program with error:%v", err), finishCbWrapper)
				break
			} else {
				log.WithFields(log.Fields{"program": p.GetName()}).Info("fail to start program with error:", err)
				p.changeStateTo(BACKOFF)
				continue
			}
		}
		if p.StdoutLog != nil {
			p.StdoutLog.SetPid(p.cmd.Process.Pid)
		}
		if p.StderrLog != nil {
			p.StderrLog.SetPid(p.cmd.Process.Pid)
		}

		monitorExited := int32(0)
		programExited := int32(0)
		//Set startsec to 0 to indicate that the program needn't stay
		//running for any particular amount of time.
		if startSecs <= 0 {
			log.WithFields(log.Fields{"program": p.GetName()}).Info("success to start program")
			p.changeStateTo(RUNNING)
			go finishCbWrapper()
		} else {
			go func() {
				p.monitorProgramIsRunning(endTime, &monitorExited, &programExited)
				finishCbWrapper()
			}()
		}
		log.WithFields(log.Fields{"program": p.GetName()}).Debug("wait program exit")
		p.lock.Unlock()
		p.waitForExit(startSecs)

		atomic.StoreInt32(&programExited, 1)
		// wait for monitor thread exit
		for atomic.LoadInt32(&monitorExited) == 0 {
			time.Sleep(time.Duration(10) * time.Millisecond)
		}

		p.lock.Lock()

		// if the program still in running after startSecs
		if p.state == RUNNING {
			p.changeStateTo(EXITED)
			log.WithFields(log.Fields{"program": p.GetName()}).Info("program exited")
			break
		} else {
			p.changeStateTo(BACKOFF)
		}

		// The number of serial failure attempts that supervisord will allow when attempting to
		// start the program before giving up and putting the process into an FATAL state
		// first start time is not the retry time
		if atomic.LoadInt32(p.retryTimes) >= p.getStartRetries() {
			p.failToStartProgram(fmt.Sprintf("fail to start program because retry times is greater than %d", p.getStartRetries()), finishCbWrapper)
			break
		}
	}

}

func (p *Process) changeStateTo(procState ProcessState) {
	if p.config.IsProgram() {
		progName := p.config.GetProgramName()
		groupName := p.config.GetGroupName()
		if procState == STARTING {
			events.EmitEvent(events.CreateProcessStartingEvent(progName, groupName, p.state.String(), int(atomic.LoadInt32(p.retryTimes))))
		} else if procState == RUNNING {
			events.EmitEvent(events.CreateProcessRunningEvent(progName, groupName, p.state.String(), p.cmd.Process.Pid))
		} else if procState == BACKOFF {
			events.EmitEvent(events.CreateProcessBackoffEvent(progName, groupName, p.state.String(), int(atomic.LoadInt32(p.retryTimes))))
		} else if procState == STOPPING {
			events.EmitEvent(events.CreateProcessStoppingEvent(progName, groupName, p.state.String(), p.cmd.Process.Pid))
		} else if procState == EXITED {
			exitCode, err := p.getExitCode()
			expected := 0
			if err == nil && p.inExitCodes(exitCode) {
				expected = 1
			}
			events.EmitEvent(events.CreateProcessExitedEvent(progName, groupName, p.state.String(), expected, p.cmd.Process.Pid))
		} else if procState == FATAL {
			events.EmitEvent(events.CreateProcessFatalEvent(progName, groupName, p.state.String()))
		} else if procState == STOPPED {
			events.EmitEvent(events.CreateProcessStoppedEvent(progName, groupName, p.state.String(), p.cmd.Process.Pid))
		} else if procState == UNKNOWN {
			events.EmitEvent(events.CreateProcessUnknownEvent(progName, groupName, p.state.String()))
		}
	}
	p.state = procState
}

// send signal to the process
//
// Args:
//   sig - the signal to the process
//   sigChildren - true: send the signal to the process and its children proess
//
func (p *Process) Signal(sig os.Signal, sigChildren bool) error {
	p.lock.RLock()
	defer p.lock.RUnlock()

	return p.sendSignal(sig, sigChildren)
}

// send signal to the process
//
// Args:
//    sig - the signal to be sent
//    sigChildren - true if the signal also need to be sent to children process
//
func (p *Process) sendSignal(sig os.Signal, sigChildren bool) error {
	if p.cmd != nil && p.cmd.Process != nil {
		err := signals.Kill(p.cmd.Process, sig, sigChildren)
		return err
	}
	return fmt.Errorf("process is not started")
}

func (p *Process) setEnv() {
	env := p.config.GetEnv("environment")
	if len(env) != 0 {
		p.cmd.Env = append(os.Environ(), env...)
	} else {
		p.cmd.Env = os.Environ()
	}
}

func (p *Process) setDir() {
	dir := p.config.GetStringExpression("directory", "")
	if dir != "" {
		p.cmd.Dir = dir
	}
}

func (p *Process) setLog() {
	if p.config.IsProgram() {
		p.StdoutLog = p.createLogger(p.GetStdoutLogfile(),
			int64(p.config.GetBytes("stdout_logfile_maxbytes", 50*1024*1024)),
			p.config.GetInt("stdout_logfile_backups", 10),
			p.createStdoutLogEventEmitter())
		capture_bytes := p.config.GetBytes("stdout_capture_maxbytes", 0)
		if capture_bytes > 0 {
			log.WithFields(log.Fields{"program": p.config.GetProgramName()}).Info("capture stdout process communication")
			p.StdoutLog = logger.NewLogCaptureLogger(p.StdoutLog,
				capture_bytes,
				"PROCESS_COMMUNICATION_STDOUT",
				p.GetName(),
				p.GetGroup())
		}

		p.cmd.Stdout = p.StdoutLog

		if p.config.GetBool("redirect_stderr", false) {
			p.StderrLog = p.StdoutLog
		} else {
			p.StderrLog = p.createLogger(p.GetStderrLogfile(),
				int64(p.config.GetBytes("stderr_logfile_maxbytes", 50*1024*1024)),
				p.config.GetInt("stderr_logfile_backups", 10),
				p.createStderrLogEventEmitter())
		}

		capture_bytes = p.config.GetBytes("stderr_capture_maxbytes", 0)

		if capture_bytes > 0 {
			log.WithFields(log.Fields{"program": p.config.GetProgramName()}).Info("capture stderr process communication")
			p.StderrLog = logger.NewLogCaptureLogger(p.StdoutLog,
				capture_bytes,
				"PROCESS_COMMUNICATION_STDERR",
				p.GetName(),
				p.GetGroup())
		}

		p.cmd.Stderr = p.StderrLog

	} else if p.config.IsEventListener() {
		in, err := p.cmd.StdoutPipe()
		if err != nil {
			log.WithFields(log.Fields{"eventListener": p.config.GetEventListenerName()}).Error("fail to get stdin")
			return
		}
		out, err := p.cmd.StdinPipe()
		if err != nil {
			log.WithFields(log.Fields{"eventListener": p.config.GetEventListenerName()}).Error("fail to get stdout")
			return
		}
		events := strings.Split(p.config.GetString("events", ""), ",")
		for i, event := range events {
			events[i] = strings.TrimSpace(event)
		}
		p.cmd.Stderr = os.Stderr

		p.registerEventListener(p.config.GetEventListenerName(),
			events,
			in,
			out)
	}
}

func (p *Process) createStdoutLogEventEmitter() logger.LogEventEmitter {
	if p.config.GetBytes("stdout_capture_maxbytes", 0) <= 0 && p.config.GetBool("stdout_events_enabled", false) {
		return logger.NewStdoutLogEventEmitter(p.config.GetProgramName(), p.config.GetGroupName(), func() int {
			return p.GetPid()
		})
	}
	return logger.NewNullLogEventEmitter()
}

func (p *Process) createStderrLogEventEmitter() logger.LogEventEmitter {
	if p.config.GetBytes("stderr_capture_maxbytes", 0) <= 0 && p.config.GetBool("stderr_events_enabled", false) {
		return logger.NewStdoutLogEventEmitter(p.config.GetProgramName(), p.config.GetGroupName(), func() int {
			return p.GetPid()
		})
	}
	return logger.NewNullLogEventEmitter()
}

func (p *Process) registerEventListener(eventListenerName string,
	_events []string,
	stdin io.Reader,
	stdout io.Writer) {
	eventListener := events.NewEventListener(eventListenerName,
		p.supervisor_id,
		stdin,
		stdout,
		p.config.GetInt("buffer_size", 100))
	events.RegisterEventListener(eventListenerName, _events, eventListener)
}

func (p *Process) unregisterEventListener(eventListenerName string) {
	events.UnregisterEventListener(eventListenerName)
}

func (p *Process) createLogger(logFile string, maxBytes int64, backups int, logEventEmitter logger.LogEventEmitter) logger.Logger {
	return logger.NewLogger(p.GetName(), logFile, logger.NewNullLocker(), maxBytes, backups, logEventEmitter)
}

func (p *Process) setUser() error {
	userName := p.config.GetString("user", "")
	if len(userName) == 0 {
		return nil
	}

	//check if group is provided
	pos := strings.Index(userName, ":")
	groupName := ""
	if pos != -1 {
		groupName = userName[pos+1:]
		userName = userName[0:pos]
	}
	u, err := user.Lookup(userName)
	if err != nil {
		return err
	}
	uid, err := strconv.ParseUint(u.Uid, 10, 32)
	if err != nil {
		return err
	}
	gid, err := strconv.ParseUint(u.Gid, 10, 32)
	if err != nil && groupName == "" {
		return err
	}
	if groupName != "" {
		g, err := user.LookupGroup(groupName)
		if err != nil {
			return err
		}
		gid, err = strconv.ParseUint(g.Gid, 10, 32)
		if err != nil {
			return err
		}
	}
	set_user_id(p.cmd.SysProcAttr, uint32(uid), uint32(gid))
	return nil
}

//send signal to process to stop it
func (p *Process) Stop(wait bool) {
	p.lock.Lock()
	p.stopByUser = true
	isRunning := p.isRunning()
	p.lock.Unlock()
	if !isRunning {
		log.WithFields(log.Fields{"program": p.GetName()}).Info("program is not running")
		return
	}
	log.WithFields(log.Fields{"program": p.GetName()}).Info("stop the program")
	sigs := strings.Fields(p.config.GetString("stopsignal", ""))
	waitsecs := time.Duration(p.config.GetInt("stopwaitsecs", 10)) * time.Second
	stopasgroup := p.config.GetBool("stopasgroup", false)
	killasgroup := p.config.GetBool("killasgroup", stopasgroup)
	if stopasgroup && !killasgroup {
		log.WithFields(log.Fields{"program": p.GetName()}).Error("Cannot set stopasgroup=true and killasgroup=false")
	}

	go func() {
		stopped := false
		for i := 0; i < len(sigs) && !stopped; i++ {
			// send signal to process
			sig, err := signals.ToSignal(sigs[i])
			if err != nil {
				continue
			}
			log.WithFields(log.Fields{"program": p.GetName(), "signal": sigs[i]}).Info("send stop signal to program")
			p.Signal(sig, stopasgroup)
			endTime := time.Now().Add(waitsecs)
			//wait at most "stopwaitsecs" seconds for one signal
			for endTime.After(time.Now()) {
				//if it already exits
				if p.state != STARTING && p.state != RUNNING && p.state != STOPPING {
					stopped = true
					break
				}
				time.Sleep(1 * time.Second)
			}
		}
		if !stopped {
			log.WithFields(log.Fields{"program": p.GetName()}).Info("force to kill the program")
			p.Signal(syscall.SIGKILL, killasgroup)
		}
	}()
	if wait {
		for {
			// if the program exits
			p.lock.RLock()
			if p.state != STARTING && p.state != RUNNING && p.state != STOPPING {
				p.lock.RUnlock()
				break
			}
			p.lock.RUnlock()
			time.Sleep(1 * time.Second)
		}
	}
}

func (p *Process) GetStatus() string {
	if p.cmd.ProcessState.Exited() {
		return p.cmd.ProcessState.String()
	}
	return "running"
}
