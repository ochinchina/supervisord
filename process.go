package main

import (
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/ochinchina/supervisord/config"
	"github.com/ochinchina/supervisord/events"
	"github.com/ochinchina/supervisord/logger"
    "github.com/ochinchina/supervisord/signals"
	"io"
	"os"
	"os/exec"
	"os/user"
	"strconv"
	"strings"
	"sync"
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
	retryTimes int
	lock       sync.RWMutex
	stdin      io.WriteCloser
	stdoutLog  logger.Logger
	stderrLog  logger.Logger
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
		retryTimes: 0}
	proc.config = config
	proc.cmd = nil

	//start the process if autostart is set to true
	//if proc.isAutoStart() {
	//	proc.Start(false)
	//}

	return proc
}

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

	var runCond *sync.Cond = nil
	finished := false
	if wait {
		runCond = sync.NewCond(&sync.Mutex{})
		runCond.L.Lock()
	}

	go func() {
		p.retryTimes = 0

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
			if (p.stopTime.Unix() - p.startTime.Unix()) < int64(p.getStartSeconds()) {
				p.retryTimes++
			} else {
				p.retryTimes = 0
			}
			if p.stopByUser {
				log.WithFields(log.Fields{"program": p.GetName()}).Info("Stopped by user, don't start it again")
				break
			}
			if !p.isAutoRestart() {
				log.WithFields(log.Fields{"program": p.GetName()}).Info("Don't start the stopped program because its autorestart flag is false")
				break
			}
			if p.retryTimes >= p.getStartRetries() {
				log.WithFields(log.Fields{"program": p.GetName()}).Info("Don't start the stopped program because its retry times ", p.retryTimes, " is greater than start retries ", p.getStartRetries())
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
	p.lock.Lock()
	defer p.lock.Unlock()
	if p.state == RUNNING {
		seconds := int(time.Now().Sub(p.startTime).Seconds())
		minutes := seconds / 60
		hours := minutes / 60
		days := hours / 24
		if days > 0 {
			return fmt.Sprintf("pid %d, uptime %d days, %d:%02d:%02d", p.cmd.Process.Pid, days, hours%24, minutes%60, seconds%60)
		} else {
			return fmt.Sprintf("pid %d, uptime %d:%02d:%02d", p.cmd.Process.Pid, hours%24, minutes%60, seconds%60)
		}
	} else if p.state != STOPPED {
		return p.stopTime.String()
	}
	return ""
}

func (p *Process) GetExitstatus() int {
	p.lock.Lock()
	defer p.lock.Unlock()

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
	p.lock.Lock()
	defer p.lock.Unlock()

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
	expand_file, err := path_expand(file_name)
	if err == nil {
		return expand_file
	} else {
		return file_name
	}
}

func (p *Process) GetStderrLogfile() string {
	file_name := p.config.GetStringExpression("stderr_logfile", "/dev/null")
	expand_file, err := path_expand(file_name)
	if err == nil {
		return expand_file
	} else {
		return file_name
	}
}

func (p *Process) getStartSeconds() int {
	return p.config.GetInt("startsecs", 1)
}

func (p *Process) getStartRetries() int {
	return p.config.GetInt("startretries", 3)
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
		p.lock.Lock()
		defer p.lock.Unlock()
		if p.cmd != nil && p.cmd.ProcessState != nil {
			exitCode, err := p.getExitCode()
			return err == nil && p.inExitCodes(exitCode)
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

func (p *Process) run(finishCb func()) {
	args, err := parseCommand(p.config.GetStringExpression("command", ""))

	if err != nil {
		log.Error("the command is empty string")
		finishCb()
		return
	}
	p.lock.Lock()
	if p.cmd != nil && p.cmd.ProcessState != nil {
		status := p.cmd.ProcessState.Sys().(syscall.WaitStatus)
		if status.Continued() {
			log.WithFields(log.Fields{"program": p.GetName()}).Info("Don't start program because it is running")
			p.lock.Unlock()
			finishCb()
			return
		}
	}
	p.cmd = exec.Command(args[0])
	if len(args) > 1 {
		p.cmd.Args = args
	}
	p.cmd.SysProcAttr = &syscall.SysProcAttr{}
	if p.setUser() != nil {
		log.WithFields(log.Fields{"user": p.config.GetString("user", "")}).Error("fail to run as user")
		p.lock.Unlock()
		finishCb()
		return
	}
	set_deathsig(p.cmd.SysProcAttr)
	p.setEnv()
	p.setDir()
	p.setLog()

	p.stdin, _ = p.cmd.StdinPipe()
	p.startTime = time.Now()
	p.changeStateTo(STARTING)
	err = p.cmd.Start()
	if err != nil {
		log.WithFields(log.Fields{"program": p.config.GetProgramName()}).Errorf("fail to start program with error:%v", err)
		p.changeStateTo(FATAL)
		p.stopTime = time.Now()
		p.lock.Unlock()
		finishCb()
	} else {
		if p.stdoutLog != nil {
			p.stdoutLog.SetPid(p.cmd.Process.Pid)
		}
		if p.stderrLog != nil {
			p.stderrLog.SetPid(p.cmd.Process.Pid)
		}
		log.WithFields(log.Fields{"program": p.config.GetProgramName()}).Info("success to start program")
		startSecs := p.config.GetInt("startsecs", 1)
		//Set startsec to 0 to indicate that the program needn't stay
		//running for any particular amount of time.
		if startSecs <= 0 {
			p.changeStateTo(RUNNING)

		} else {
			time.Sleep(time.Duration(startSecs) * time.Second)
			if tmpProc, err := os.FindProcess(p.cmd.Process.Pid); err == nil && tmpProc != nil {
				p.changeStateTo(RUNNING)
			}
		}
		p.lock.Unlock()
		log.WithFields(log.Fields{"program": p.config.GetProgramName()}).Debug("wait program exit")
		finishCb()
		err = p.cmd.Wait()
		if err == nil {
			if p.cmd.ProcessState != nil {
				log.WithFields(log.Fields{"program": p.config.GetProgramName()}).Infof("program stopped with status:%v", p.cmd.ProcessState)
			} else {
				log.WithFields(log.Fields{"program": p.config.GetProgramName()}).Info("program stopped")
			}
		} else {
			log.WithFields(log.Fields{"program": p.config.GetProgramName()}).Errorf("program stopped with error:%v", err)
		}

		p.lock.Lock()
		p.stopTime = time.Now()
		if p.stopTime.Unix()-p.startTime.Unix() < int64(startSecs) {
			p.changeStateTo(BACKOFF)
		} else {
			p.changeStateTo(EXITED)
		}
		p.lock.Unlock()
	}

}

func (p *Process) changeStateTo(procState ProcessState) {
	if p.config.IsProgram() {
		progName := p.config.GetProgramName()
		groupName := p.config.GetGroupName()
		if procState == STARTING {
			events.EmitEvent(events.CreateProcessStartingEvent(progName, groupName, p.state.String(), p.retryTimes))
		} else if procState == RUNNING {
			events.EmitEvent(events.CreateProcessRunningEvent(progName, groupName, p.state.String(), p.cmd.Process.Pid))
		} else if procState == BACKOFF {
			events.EmitEvent(events.CreateProcessBackoffEvent(progName, groupName, p.state.String(), p.retryTimes))
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

func (p *Process) Signal(sig os.Signal) error {
	p.lock.Lock()
	defer p.lock.Unlock()

	return p.sendSignal(sig)
}

func (p *Process) sendSignal(sig os.Signal) error {
	if p.cmd != nil && p.cmd.Process != nil {
		err := signals.Kill(p.cmd.Process, sig)
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
		p.stdoutLog = p.createLogger(p.GetStdoutLogfile(),
			int64(p.config.GetBytes("stdout_logfile_maxbytes", 50*1024*1024)),
			p.config.GetInt("stdout_logfile_backups", 10),
			p.createStdoutLogEventEmitter())
		capture_bytes := p.config.GetBytes("stdout_capture_maxbytes", 0)
		if capture_bytes > 0 {
			log.WithFields(log.Fields{"program": p.config.GetProgramName()}).Info("capture stdout process communication")
			p.stdoutLog = logger.NewLogCaptureLogger(p.stdoutLog,
				capture_bytes,
				"PROCESS_COMMUNICATION_STDOUT",
				p.GetName(),
				p.GetGroup())
		}

		p.cmd.Stdout = p.stdoutLog

		if p.config.GetBool("redirect_stderr", false) {
			p.stderrLog = p.stdoutLog
		} else {
			p.stderrLog = p.createLogger(p.GetStderrLogfile(),
				int64(p.config.GetBytes("stderr_logfile_maxbytes", 50*1024*1024)),
				p.config.GetInt("stderr_logfile_backups", 10),
				p.createStderrLogEventEmitter())
		}

		capture_bytes = p.config.GetBytes("stderr_capture_maxbytes", 0)

		if capture_bytes > 0 {
			log.WithFields(log.Fields{"program": p.config.GetProgramName()}).Info("capture stderr process communication")
			p.stderrLog = logger.NewLogCaptureLogger(p.stdoutLog,
				capture_bytes,
				"PROCESS_COMMUNICATION_STDERR",
				p.GetName(),
				p.GetGroup())
		}

		p.cmd.Stderr = p.stderrLog

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
	} else {
		return logger.NewNullLogEventEmitter()
	}
}

func (p *Process) createStderrLogEventEmitter() logger.LogEventEmitter {
	if p.config.GetBytes("stderr_capture_maxbytes", 0) <= 0 && p.config.GetBool("stderr_events_enabled", false) {
		return logger.NewStdoutLogEventEmitter(p.config.GetProgramName(), p.config.GetGroupName(), func() int {
			return p.GetPid()
		})
	} else {
		return logger.NewNullLogEventEmitter()
	}
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
	var mylogger logger.Logger
	mylogger = logger.NewNullLogger(logEventEmitter)

	if logFile == "/dev/stdout" {
		mylogger = logger.NewStdoutLogger(logEventEmitter)
	} else if logFile == "/dev/stderr" {
		mylogger = logger.NewStderrLogger(logEventEmitter)
	} else if logFile == "syslog" {
		mylogger = logger.NewSysLogger(p.GetName(), logEventEmitter)
	} else if len(logFile) > 0 {
		mylogger = logger.NewFileLogger(logFile, maxBytes, backups, logEventEmitter, logger.NewNullLocker())
	}
	return mylogger
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
	p.lock.RLock()
	p.stopByUser = true
	p.lock.RUnlock()
	log.WithFields(log.Fields{"program": p.GetName()}).Info("stop the program")
	sig, err := signals.ToSignal(p.config.GetString("stopsignal", ""))
	if err == nil {
		p.Signal(sig)
	}
	waitsecs := time.Duration(p.config.GetInt("stopwaitsecs", 10)) * time.Second
	endTime := time.Now().Add(waitsecs)
	go func() {
		//wait at most "stopwaitsecs" seconds
		for {
			//if it already exits
			if p.state != STARTING && p.state != RUNNING && p.state != STOPPING {
				break
			}
			//if endTime reaches, raise signal syscall.SIGKILL
			if endTime.Before(time.Now()) {
				log.WithFields(log.Fields{"program": p.GetName()}).Info("force to kill the program")
				p.Signal(syscall.SIGKILL)
				break
			} else {
				time.Sleep(1 * time.Second)
			}
		}
	}()
	if wait {
		for {
			// if the program exits
			if p.state != STARTING && p.state != RUNNING && p.state != STOPPING {
				break
			}
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
