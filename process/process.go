package process

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/mitchellh/go-ps"
	"github.com/ochinchina/filechangemonitor"
	"github.com/ochinchina/supervisord/config"
	"github.com/ochinchina/supervisord/events"
	"github.com/ochinchina/supervisord/logger"
	"github.com/ochinchina/supervisord/signals"
	"github.com/robfig/cron/v3"
	log "github.com/sirupsen/logrus"
)

// State the state of process
type State int

const (
	// Stopped the stopped state
	Stopped State = iota

	// Starting the starting state
	Starting = 10

	// Running the running state
	Running = 20

	// Backoff the backoff state
	Backoff = 30

	// Stopping the stopping state
	Stopping = 40

	// Exited the Exited state
	Exited = 100

	// Fatal the Fatal state
	Fatal = 200

	// Unknown the unknown state
	Unknown = 1000
)

var scheduler *cron.Cron = nil

func init() {
	scheduler = cron.New(cron.WithSeconds())
	scheduler.Start()
}

// String convert State to human-readable string
func (p State) String() string {
	switch p {
	case Stopped:
		return "Stopped"
	case Starting:
		return "Starting"
	case Running:
		return "Running"
	case Backoff:
		return "Backoff"
	case Stopping:
		return "Stopping"
	case Exited:
		return "Exited"
	case Fatal:
		return "Fatal"
	default:
		return "Unknown"
	}
}

// Process the program process management data
type Process struct {
	supervisorID string
	config       *config.Entry
	cmd          *exec.Cmd
	startTime    time.Time
	stopTime     time.Time
	state        State
	// true if process is starting
	inStart bool
	// true if the process is stopped by user
	stopByUser bool
	retryTimes *int32
	lock       sync.RWMutex
	stdin      io.WriteCloser
	StdoutLog  logger.Logger
	StderrLog  logger.Logger
}

// NewProcess creates new Process object
func NewProcess(supervisorID string, config *config.Entry) *Process {
	proc := &Process{supervisorID: supervisorID,
		config:     config,
		cmd:        nil,
		startTime:  time.Unix(0, 0),
		stopTime:   time.Unix(0, 0),
		state:      Stopped,
		inStart:    false,
		stopByUser: false,
		retryTimes: new(int32)}
	proc.config = config
	proc.cmd = nil
	proc.addToCron()
	return proc
}

func (p *Process) GetConfig() *config.Entry {
	return p.config
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

// Start process
// Args:
//
//	wait - true, wait the program started or failed
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
	if wait {
		runCond = sync.NewCond(&sync.Mutex{})
		runCond.L.Lock()
	}

	go func() {
		for {
			// we'll do retry start if it sets.
			p.run(func() {
				if wait {
					runCond.L.Lock()
					runCond.Signal()
					runCond.L.Unlock()
				}
			})
			// avoid print too many logs if fail to start program too quickly
			if time.Now().Unix()-p.startTime.Unix() < 2 {
				time.Sleep(5 * time.Second)
			}
			if p.stopByUser {
				log.WithFields(log.Fields{"program": p.GetName()}).Info("program stopped by user, don't start it again")
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

	if wait {
		runCond.Wait()
		runCond.L.Unlock()
	}
}

// GetName returns name of program or event listener
func (p *Process) GetName() string {
	if p.config.IsProgram() {
		return p.config.GetProgramName()
	} else if p.config.IsEventListener() {
		return p.config.GetEventListenerName()
	} else {
		return ""
	}
}

// GetGroup returns group the program belongs to
func (p *Process) GetGroup() string {
	return p.config.Group
}

// GetDescription returns process status description
func (p *Process) GetDescription() string {
	p.lock.RLock()
	defer p.lock.RUnlock()
	if p.state == Running {
		seconds := int(time.Now().Sub(p.startTime).Seconds())
		minutes := seconds / 60
		hours := minutes / 60
		days := hours / 24
		if days > 0 {
			return fmt.Sprintf("pid %d, uptime %d days, %d:%02d:%02d", p.cmd.Process.Pid, days, hours%24, minutes%60, seconds%60)
		}
		return fmt.Sprintf("pid %d, uptime %d:%02d:%02d", p.cmd.Process.Pid, hours%24, minutes%60, seconds%60)
	} else if p.state != Stopped {
                if p.stopTime.Unix() > 0 {
			return p.stopTime.String()
		}
	}
	return ""
}

// GetExitstatus returns exit status of the process if the program exit
func (p *Process) GetExitstatus() int {
	p.lock.RLock()
	defer p.lock.RUnlock()

	if p.state == Exited || p.state == Backoff {
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

// GetPid returns pid of running process or 0 it is not in running status
func (p *Process) GetPid() int {
	p.lock.RLock()
	defer p.lock.RUnlock()

	if p.state == Stopped || p.state == Fatal || p.state == Unknown || p.state == Exited || p.state == Backoff {
		return 0
	}
	return p.cmd.Process.Pid
}

// GetState returns process state
func (p *Process) GetState() State {
	return p.state
}

// GetStartTime returns process start time
func (p *Process) GetStartTime() time.Time {
	return p.startTime
}

// GetStopTime returns process stop time
func (p *Process) GetStopTime() time.Time {
	switch p.state {
	case Starting:
		fallthrough
	case Running:
		fallthrough
	case Stopping:
		return time.Unix(0, 0)
	default:
		return p.stopTime
	}
}

// GetStdoutLogfile returns program stdout log filename
func (p *Process) GetStdoutLogfile() string {
	fileName := p.config.GetStringExpression("stdout_logfile", "/dev/null")
	expandFile, err := PathExpand(fileName)
	if err != nil {
		return fileName
	}
	return expandFile
}

// GetStderrLogfile returns program stderr log filename
func (p *Process) GetStderrLogfile() string {
	fileName := p.config.GetStringExpression("stderr_logfile", "/dev/null")
	expandFile, err := PathExpand(fileName)
	if err != nil {
		return fileName
	}
	return expandFile
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

// GetPriority returns program priority (as it set in config) with default value of 999
func (p *Process) GetPriority() int {
	return p.config.GetInt("priority", 999)
}

func (p *Process) getNumberProcs() int {
	return p.config.GetInt("numprocs", 1)
}

// SendProcessStdin sends data to process stdin
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
			// If unexpected, the process will be restarted when the program exits
			// with an exit code that is not one of the exit codes associated with
			// this process’ configuration (see exitcodes).
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
func (p *Process) isRunning() bool {
	if p.cmd != nil && p.cmd.Process != nil {
		if runtime.GOOS == "windows" {
			proc, err := ps.FindProcess(p.cmd.Process.Pid)
			return proc != nil && err == nil
		}
		return p.cmd.Process.Signal(syscall.Signal(0)) == nil
	}
	return false
}

// create Command object for the program
func (p *Process) createProgramCommand() error {
	args, err := parseCommand(p.config.GetStringExpression("command", ""))

	if err != nil {
		return err
	}
	p.cmd, err = createCommand(args)
	if err != nil {
		return err
	}
	if p.setUser() != nil {
		log.WithFields(log.Fields{"user": p.config.GetString("user", "")}).Error("fail to run as user")
		return fmt.Errorf("fail to set user")
	}
	p.setProgramRestartChangeMonitor(args[0])
	setDeathsig(p.cmd.SysProcAttr)
	p.setEnv()
	p.setDir()
	p.setLog()

	p.stdin, _ = p.cmd.StdinPipe()
	return nil

}

func (p *Process) setProgramRestartChangeMonitor(programPath string) {
	if p.config.GetBool("restart_when_binary_changed", false) {
		absPath, err := filepath.Abs(programPath)
		if err != nil {
			absPath = programPath
		}
		AddProgramChangeMonitor(absPath, func(path string, mode filechangemonitor.FileChangeMode) {
			log.WithFields(log.Fields{"program": p.GetName()}).Info("program is changed, restart it")
			restart_cmd := p.config.GetString("restart_cmd_when_binary_changed", "")
			s := p.config.GetString("restart_signal_when_binary_changed", "")
			if len(restart_cmd) > 0 {
				_, err := executeCommand(restart_cmd)
				if err == nil {
					log.WithFields(log.Fields{"program": p.GetName(), "command": restart_cmd}).Info("restart program with command successfully")
				} else {
					log.WithFields(log.Fields{"program": p.GetName(), "command": restart_cmd, "error": err}).Info("fail to restart program")
				}
			} else if len(s) > 0 {
				p.sendSignals(strings.Fields(s), true)
			} else {
				p.Stop(true)
				p.Start(true)
			}

		})
	}
	dirMonitor := p.config.GetString("restart_directory_monitor", "")
	filePattern := p.config.GetString("restart_file_pattern", "*")
	if dirMonitor != "" {
		absDir, err := filepath.Abs(dirMonitor)
		if err != nil {
			absDir = dirMonitor
		}
		AddConfigChangeMonitor(absDir, filePattern, func(path string, mode filechangemonitor.FileChangeMode) {
			log.WithFields(log.Fields{"program": p.GetName()}).Info("configure file for program is changed, restart it")
			restart_cmd := p.config.GetString("restart_cmd_when_file_changed", "")
			s := p.config.GetString("restart_signal_when_file_changed", "")
			if len(restart_cmd) > 0 {
				_, err := executeCommand(restart_cmd)
				if err == nil {
					log.WithFields(log.Fields{"program": p.GetName(), "command": restart_cmd}).Info("restart program with command successfully")
				} else {
					log.WithFields(log.Fields{"program": p.GetName(), "command": restart_cmd, "error": err}).Info("fail to restart program")
				}
			} else if len(s) > 0 {
				p.sendSignals(strings.Fields(s), true)
			} else {
				p.Stop(true)
				p.Start(true)
			}
		})
	}

}

// wait for the started program exit
func (p *Process) waitForExit(startSecs int64) {
	p.cmd.Wait()
	if p.cmd.ProcessState != nil {
		log.WithFields(log.Fields{"program": p.GetName()}).Infof("program stopped with status:%v", p.cmd.ProcessState)
	} else {
		log.WithFields(log.Fields{"program": p.GetName()}).Info("program stopped")
	}
	p.lock.Lock()
	defer p.lock.Unlock()
	p.stopTime = time.Now()

	// FIXME: we didn't set eventlistener logger
	// since it's stdout/stderr has been specifically managed.
	if p.StdoutLog != nil {
		p.StdoutLog.Close()
	}
	if p.StderrLog != nil {
		p.StderrLog.Close()
	}

}

// fail to start the program
func (p *Process) failToStartProgram(reason string, finishCb func()) {
	log.WithFields(log.Fields{"program": p.GetName()}).Errorf(reason)
	p.changeStateTo(Fatal)
	finishCb()
}

// monitor if the program is in running before endTime
func (p *Process) monitorProgramIsRunning(endTime time.Time, monitorExited *int32, programExited *int32) {
	// if time is not expired
	for time.Now().Before(endTime) && atomic.LoadInt32(programExited) == 0 {
		time.Sleep(time.Duration(100) * time.Millisecond)
	}
	atomic.StoreInt32(monitorExited, 1)

	p.lock.Lock()
	defer p.lock.Unlock()
	// if the program does not exit
	if atomic.LoadInt32(programExited) == 0 && p.state == Starting {
		log.WithFields(log.Fields{"program": p.GetName()}).Info("success to start program")
		p.changeStateTo(Running)
	}
}

// 这个函数可能有以下几种执行完成的情况：
//
// 1. 程序正在运行中，因此函数直接返回。
// 2. 程序尚未运行，函数开始尝试多次启动程序，直到启动成功。
// 3. 程序成功启动并正在运行中，函数启动了一个后台监视程序来监视程序运行情况，并向 `finishCb` 函数传递一个标记告知程序已停止，函数直接返回。
// 4. 程序启动失败，超出了尝试次数，函数将程序状态标记为 `FATAL`，并向 `finishCb` 函数传递一个标记告知程序已停止，函数直接返回。
// 5. 程序被终止或运行失败，超出了重试次数，函数将程序状态标记为 `EXITED`，并向 `finishCb` 函数传递一个标记告知程序已停止，函数直接返回。
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
			// pause
			p.lock.Unlock()
			log.WithFields(log.Fields{"program": p.GetName()}).Info("don't restart the program, start it after ", restartPause, " seconds")
			time.Sleep(time.Duration(restartPause) * time.Second)
			p.lock.Lock()
		}
		endTime := time.Now().Add(time.Duration(startSecs) * time.Second)
		p.changeStateTo(Starting)
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
				p.changeStateTo(Backoff)
				continue
			}
		}
		if p.StdoutLog != nil {
			p.StdoutLog.SetPid(p.cmd.Process.Pid)
		}
		if p.StderrLog != nil {
			p.StderrLog.SetPid(p.cmd.Process.Pid)
		}

		// logger.CompositeLogger is not `os.File`, so `cmd.Wait()` will wait for the logger to close
		// if parent process passes its FD to child process, the logger will not close even when parent process exits
		// we need to make sure the logger is closed when the process stops running
		go func() {
			// the sleep time must be less than `stopwaitsecs`, here I set half of `stopwaitsecs`
			// otherwise the logger will not be closed before SIGKILL is sent
			halfWaitsecs := time.Duration(p.config.GetInt("stopwaitsecs", 10)/2) * time.Second
			for {
				if !p.isRunning() {
					break
				}
				time.Sleep(halfWaitsecs)
			}
			if p.StdoutLog != nil {
				p.StdoutLog.Close()
			}
			if p.StderrLog != nil {
				p.StderrLog.Close()
			}
		}()

		monitorExited := int32(0)
		programExited := int32(0)
		// Set startsec to 0 to indicate that the program needn't stay
		// running for any particular amount of time.
		if startSecs <= 0 {
			atomic.StoreInt32(&monitorExited, 1)
			log.WithFields(log.Fields{"program": p.GetName()}).Info("success to start program")
			p.changeStateTo(Running)
			go finishCbWrapper()
		} else {
			go func() {
				p.monitorProgramIsRunning(endTime, &monitorExited, &programExited)
				finishCbWrapper()
			}()
		}
		log.WithFields(log.Fields{"program": p.GetName()}).Debug("check program is starting and wait if it exit")
		p.lock.Unlock()

		procExitC := make(chan struct{})
		go func() {
			p.waitForExit(startSecs)
			close(procExitC)
		}()

	LOOP:
		for {
			select {
			case <-procExitC:
				break LOOP
			default:
				if !p.isRunning() {
					break LOOP
				}
			}
			time.Sleep(time.Duration(100) * time.Millisecond)
		}

		atomic.StoreInt32(&programExited, 1)
		// wait for monitor thread exit
		for atomic.LoadInt32(&monitorExited) == 0 {
			time.Sleep(time.Duration(10) * time.Millisecond)
		}

		p.lock.Lock()

		// we break the restartRetry loop if:
		// 1. process still in running after startSecs (although it's exited right now)
		// 2. it's stopping by user (we unlocked before waitForExit, so the flag stopByUser will have a chance to change).
		if p.state == Running || p.state == Stopping {
			if !p.stopByUser {
				p.changeStateTo(Exited)
				log.WithFields(log.Fields{"program": p.GetName()}).Info("program exited")
			} else {
				p.changeStateTo(Stopped)
				log.WithFields(log.Fields{"program": p.GetName()}).Info("program stopped by user")
			}
			break
		} else {
			p.changeStateTo(Backoff)
		}

		// The number of serial failure attempts that supervisord will allow when attempting to
		// start the program before giving up and putting the process into an Fatal state
		// first start time is not the retry time
		if atomic.LoadInt32(p.retryTimes) >= p.getStartRetries() {
			p.failToStartProgram(fmt.Sprintf("fail to start program because retry times is greater than %d", p.getStartRetries()), finishCbWrapper)
			break
		}
	}

}

func (p *Process) changeStateTo(procState State) {
	if p.config.IsProgram() {
		progName := p.config.GetProgramName()
		groupName := p.config.GetGroupName()
		if procState == Starting {
			events.EmitEvent(events.CreateProcessStartingEvent(progName, groupName, p.state.String(), int(atomic.LoadInt32(p.retryTimes))))
		} else if procState == Running {
			events.EmitEvent(events.CreateProcessRunningEvent(progName, groupName, p.state.String(), p.cmd.Process.Pid))
		} else if procState == Backoff {
			events.EmitEvent(events.CreateProcessBackoffEvent(progName, groupName, p.state.String(), int(atomic.LoadInt32(p.retryTimes))))
		} else if procState == Stopping {
			events.EmitEvent(events.CreateProcessStoppingEvent(progName, groupName, p.state.String(), p.cmd.Process.Pid))
		} else if procState == Exited {
			exitCode, err := p.getExitCode()
			expected := 0
			if err == nil && p.inExitCodes(exitCode) {
				expected = 1
			}
			events.EmitEvent(events.CreateProcessExitedEvent(progName, groupName, p.state.String(), expected, p.cmd.Process.Pid))
		} else if procState == Fatal {
			events.EmitEvent(events.CreateProcessFatalEvent(progName, groupName, p.state.String()))
		} else if procState == Stopped {
			events.EmitEvent(events.CreateProcessStoppedEvent(progName, groupName, p.state.String(), p.cmd.Process.Pid))
		} else if procState == Unknown {
			events.EmitEvent(events.CreateProcessUnknownEvent(progName, groupName, p.state.String()))
		}
	}
	p.state = procState
}

// Signal sends signal to the process
//
// Args:
//
//	sig - the signal to the process
//	sigChildren - if true, sends the same signal to the process and its children
func (p *Process) Signal(sig os.Signal, sigChildren bool) error {
	p.lock.RLock()
	defer p.lock.RUnlock()

	return p.sendSignal(sig, sigChildren)
}

func (p *Process) sendSignals(sigs []string, sigChildren bool) {
	p.lock.RLock()
	defer p.lock.RUnlock()

	for _, strSig := range sigs {
		sig, err := signals.ToSignal(strSig)
		if err == nil {
			p.sendSignal(sig, sigChildren)
		} else {
			log.WithFields(log.Fields{"program": p.GetName(), "signal": strSig}).Info("Invalid signal name")
		}
	}
}

// send signal to the process
//
// Args:
//
//	sig - the signal to be sent
//	sigChildren - if true, the signal also will be sent to children processes too
func (p *Process) sendSignal(sig os.Signal, sigChildren bool) error {
	if p.cmd != nil && p.cmd.Process != nil {
		log.WithFields(log.Fields{"program": p.GetName(), "signal": sig}).Info("Send signal to program")
		err := signals.Kill(p.cmd.Process, sig, sigChildren)
		return err
	}
	return fmt.Errorf("process is not started")
}

func (p *Process) setEnv() {
	envFromFiles := p.config.GetEnvFromFiles("envFiles")
	env := p.config.GetEnv("environment")
	if len(env)+len(envFromFiles) != 0 {
		p.cmd.Env = mergeKeyValueArrays(p.cmd.Env, append(append(os.Environ(), envFromFiles...), env...))
	} else {
		p.cmd.Env = mergeKeyValueArrays(p.cmd.Env, os.Environ())
	}
}

// 辅助函数：带覆盖的环境变量追加
func mergeKeyValueArrays(arr1, arr2 []string) []string {
	keySet := make(map[string]bool)
	result := make([]string, 0, len(arr1)+len(arr2))

	// 处理第一个数组，保留所有元素
	for _, item := range arr1 {
		if key := strings.SplitN(item, "=", 2)[0]; key != "" {
			keySet[key] = true
		}
		result = append(result, item)
	}

	// 处理第二个数组，跳过已存在的键
	for _, item := range arr2 {
		if key := strings.SplitN(item, "=", 2)[0]; key != "" {
			if !keySet[key] {
				result = append(result, item)
			}
		}
	}

	return result
}

func (p *Process) setDir() {
	dir := p.config.GetStringExpression("directory", "")
	if dir != "" {
		p.cmd.Dir = dir
	}
}

func (p *Process) setLog() {
	if p.config.IsProgram() {
		p.StdoutLog = p.createStdoutLogger()
		captureBytes := p.config.GetBytes("stdout_capture_maxbytes", 0)
		if captureBytes > 0 {
			log.WithFields(log.Fields{"program": p.config.GetProgramName()}).Info("capture stdout process communication")
			p.StdoutLog = logger.NewLogCaptureLogger(p.StdoutLog,
				captureBytes,
				"PROCESS_COMMUNICATION_STDOUT",
				p.GetName(),
				p.GetGroup())
		}

		p.cmd.Stdout = p.StdoutLog

		if p.config.GetBool("redirect_stderr", false) {
			p.StderrLog = p.StdoutLog
		} else {
			p.StderrLog = p.createStderrLogger()
		}

		captureBytes = p.config.GetBytes("stderr_capture_maxbytes", 0)

		if captureBytes > 0 {
			log.WithFields(log.Fields{"program": p.config.GetProgramName()}).Info("capture stderr process communication")
			p.StderrLog = logger.NewLogCaptureLogger(p.StdoutLog,
				captureBytes,
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
		p.supervisorID,
		stdin,
		stdout,
		p.config.GetInt("buffer_size", 100))
	events.RegisterEventListener(eventListenerName, _events, eventListener)
}

func (p *Process) unregisterEventListener(eventListenerName string) {
	events.UnregisterEventListener(eventListenerName)
}

func (p *Process) createStdoutLogger() logger.Logger {
	logFile := p.GetStdoutLogfile()
	maxBytes := int64(p.config.GetBytes("stdout_logfile_maxbytes", 50*1024*1024))
	backups := p.config.GetInt("stdout_logfile_backups", 10)
	logEventEmitter := p.createStdoutLogEventEmitter()
	props := make(map[string]string)
	syslog_facility := p.config.GetString("syslog_facility", "")
	syslog_tag := p.config.GetString("syslog_tag", "")
	syslog_priority := p.config.GetString("syslog_stdout_priority", "")

	if len(syslog_facility) > 0 {
		props["syslog_facility"] = syslog_facility
	}
	if len(syslog_tag) > 0 {
		props["syslog_tag"] = syslog_tag
	}
	if len(syslog_priority) > 0 {
		props["syslog_priority"] = syslog_priority
	}

	return logger.NewLogger(p.GetName(), logFile, logger.NewNullLocker(), maxBytes, backups, props, logEventEmitter)
}

func (p *Process) createStderrLogger() logger.Logger {
	logFile := p.GetStderrLogfile()
	maxBytes := int64(p.config.GetBytes("stderr_logfile_maxbytes", 50*1024*1024))
	backups := p.config.GetInt("stderr_logfile_backups", 10)
	logEventEmitter := p.createStderrLogEventEmitter()
	props := make(map[string]string)
	syslog_facility := p.config.GetString("syslog_facility", "")
	syslog_tag := p.config.GetString("syslog_tag", "")
	syslog_priority := p.config.GetString("syslog_stderr_priority", "")

	if len(syslog_facility) > 0 {
		props["syslog_facility"] = syslog_facility
	}
	if len(syslog_tag) > 0 {
		props["syslog_tag"] = syslog_tag
	}
	if len(syslog_priority) > 0 {
		props["syslog_priority"] = syslog_priority
	}

	return logger.NewLogger(p.GetName(), logFile, logger.NewNullLocker(), maxBytes, backups, props, logEventEmitter)
}

func (p *Process) setUser() error {
	userName := p.config.GetString("user", "")
	if len(userName) == 0 {
		return nil
	}

	// check if group is provided
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
	setUserID(p.cmd.SysProcAttr, uint32(uid), uint32(gid))

	// 强制设置关键环境变量
	p.cmd.Env = appendEnvWithOverride(p.cmd.Env,
		"HOME", u.HomeDir, // 强制HOME目录
		"USER", u.Username, // 用户名
		"LOGNAME", u.Username, // 登录名
		"PATH", defaultPath(u), // 安全PATH
	)

	// 删除root残留的环境变量
	filterRootEnv(&p.cmd.Env)

	return nil
}

// 辅助函数：带覆盖的环境变量追加
func appendEnvWithOverride(env []string, pairs ...string) []string {
	newEnv := make([]string, 0, len(env)+len(pairs)/2)
	set := make(map[string]bool)

	// 先添加新变量
	for i := 0; i < len(pairs); i += 2 {
		key := pairs[i]
		value := pairs[i+1]
		newEnv = append(newEnv, fmt.Sprintf("%s=%s", key, value))
		set[key] = true
	}

	// 保留未覆盖的旧变量
	for _, e := range env {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) < 2 || set[parts[0]] {
			continue
		}
		newEnv = append(newEnv, e)
	}

	return newEnv
}

// 辅助函数：生成安全PATH
func defaultPath(u *user.User) string {
	// 根据用户类型返回不同PATH
	if u.Uid == "0" {
		return "/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"
	}
	return "/usr/local/bin:/usr/bin:/bin:/usr/local/games:/usr/games"
}

// 辅助函数：过滤危险的环境变量
func filterRootEnv(env *[]string) {
	filtered := make([]string, 0, len(*env))
	for _, e := range *env {
		if strings.HasPrefix(e, "SUDO_") ||
			strings.HasPrefix(e, "XDG_RUNTIME_DIR=") {
			continue
		}
		filtered = append(filtered, e)
	}
	*env = filtered
}

// Stop sends signal to process to make it quit
func (p *Process) Stop(wait bool) {
	p.lock.Lock()
	p.stopByUser = true
	isRunning := p.isRunning()
	p.lock.Unlock()
	if !isRunning {
		log.WithFields(log.Fields{"program": p.GetName()}).Info("program is not running")
		return
	}
  
	log.WithFields(log.Fields{"program": p.GetName()}).Info("stopping the program")
	p.changeStateTo(Stopping)
	sigs := strings.Fields(p.config.GetString("stopsignal", "TERM"))
	waitsecs := time.Duration(p.config.GetInt("stopwaitsecs", 10)) * time.Second
	killwaitsecs := time.Duration(p.config.GetInt("killwaitsecs", 2)) * time.Second
	stopasgroup := p.config.GetBool("stopasgroup", false)
	killasgroup := p.config.GetBool("killasgroup", stopasgroup)
	if stopasgroup && !killasgroup {
		log.WithFields(log.Fields{"program": p.GetName()}).Error("Cannot set stopasgroup=true and killasgroup=false")
	}

	var stopped int32 = 0
	go func() {
		for i := 0; i < len(sigs) && atomic.LoadInt32(&stopped) == 0; i++ {
			// send signal to process
			sig, err := signals.ToSignal(sigs[i])
			if err != nil {
				continue
			}
			log.WithFields(log.Fields{"program": p.GetName(), "signal": sigs[i]}).Info("send stop signal to program")
			p.Signal(sig, stopasgroup)
			endTime := time.Now().Add(waitsecs)
			// wait at most "stopwaitsecs" seconds for one signal
			for endTime.After(time.Now()) {
				// if it already exits
				if p.state != Starting && p.state != Running && p.state != Stopping {
					atomic.StoreInt32(&stopped, 1)
					break
				}
				time.Sleep(10 * time.Millisecond)
			}
		}
		if atomic.LoadInt32(&stopped) == 0 {
			log.WithFields(log.Fields{"program": p.GetName()}).Info("force to kill the program")
			p.Signal(syscall.SIGKILL, killasgroup)
			killEndTime := time.Now().Add(killwaitsecs)
			for killEndTime.After(time.Now()) {
				// if it exits
				if p.state != Starting && p.state != Running && p.state != Stopping {
					atomic.StoreInt32(&stopped, 1)
					break
				}
				time.Sleep(10 * time.Millisecond)
			}
			atomic.StoreInt32(&stopped, 1)
		}
	}()
	if wait {
		for atomic.LoadInt32(&stopped) == 0 {
			time.Sleep(1 * time.Second)
		}
	}
}

// GetStatus returns status of program as a string
func (p *Process) GetStatus() string {
	if p.cmd.ProcessState == nil {
		return "<nil>"
	}
	if p.cmd.ProcessState.Exited() {
		return p.cmd.ProcessState.String()
	}
	return "running"
}
