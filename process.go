package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/user"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	log "github.com/Sirupsen/logrus"
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
	config        *ConfigEntry
	cmd           *exec.Cmd
	startTime     time.Time
	stopTime      time.Time
	state         ProcessState
	//true if process is starting
	inStart bool
	//true if the process is stopped by user
	stopByUser bool
	lock       sync.RWMutex
	stdin      io.WriteCloser
	stdoutLog  Logger
	stderrLog  Logger
}

type ProcessSortByPriority []*Process

func NewProcess(supervisor_id string, config *ConfigEntry) *Process {
	proc := &Process{supervisor_id: supervisor_id,
		config:    config,
		cmd:       nil,
		startTime: time.Unix(0, 0),
		stopTime:  time.Unix(0, 0),
		state:     STOPPED}
	proc.config = config
	proc.cmd = nil
	proc.state = STOPPED
	proc.inStart = false
	proc.stopByUser = false

	//start the process if autostart is set to true
	if proc.isAutoStart() {
		proc.Start(false)
	}

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

	var m sync.Mutex
	runCond := sync.NewCond(&m)
	m.Lock()

	go func() {
		retryTimes := 0

		for {
			p.run(runCond)
			if (p.stopTime.Unix() - p.startTime.Unix()) < int64(p.getStartSeconds()) {
				retryTimes++
			} else {
				retryTimes = 0
			}
			if p.stopByUser {
				log.WithFields(log.Fields{"program": p.GetName()}).Info("Stopped by user, don't start it again")
				break
			}
			if !p.isAutoRestart() {
				log.WithFields(log.Fields{"program": p.GetName()}).Info("Don't start the stopped program because its autorestart flag is false")
				break
			}
			if retryTimes >= p.getStartRetries() {
				log.WithFields(log.Fields{"program": p.GetName()}).Info("Don't start the stopped program because its retry times ", retryTimes, " is greater than start retries ", p.getStartRetries())
				break
			}
		}
		p.lock.Lock()
		p.inStart = false
		p.lock.Unlock()
	}()
	if wait {
		runCond.Wait()
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
	if p.state == RUNNING {
		d := time.Now().Sub(p.startTime)
		return fmt.Sprintf("pid %d, uptime %02d:%02d:%02d", p.cmd.Process.Pid, int(d.Hours()), int(d.Minutes()), int(d.Seconds()))
	} else if p.state != STOPPED {
		return p.stopTime.String()
	}
	return ""
}

func (p *Process) GetExitstatus() int {
	if p.state == EXITED || p.state == BACKOFF {
		status, ok := p.cmd.ProcessState.Sys().(syscall.WaitStatus)
		if ok {
			return status.ExitStatus()
		}
	}
	return 0
}

func (p *Process) GetPid() int {
	if p.state == STOPPED {
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
	return p.config.GetString("stdout_logfile", "/dev/null")
}

func (p *Process) GetStderrLogfile() string {
	return p.config.GetString("stderr_logfile", "/dev/null")
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
			if status, ok := p.cmd.ProcessState.Sys().(syscall.WaitStatus); ok && !p.inExitCodes(status.ExitStatus()) {
				return true
			}
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

func (p *Process) run(runCond *sync.Cond) {
	args, err := parseCommand(p.config.GetString("command", ""))

	if err != nil {
		log.Error("the command is empty string")
		return
	}
	p.lock.Lock()
	if p.cmd != nil {
		status := p.cmd.ProcessState.Sys().(syscall.WaitStatus)
		if status.Continued() {
			log.WithFields(log.Fields{"program": p.GetName()}).Info("Don't start program because it is running")
			p.lock.Unlock()
			return
		}
	}
	p.cmd = exec.Command(args[0])
	if len(args) > 1 {
		p.cmd.Args = args
	}
	if p.setUser() != nil {
		log.WithFields(log.Fields{"user": p.config.GetString("user", "")}).Error("fail to run as user")
		p.lock.Unlock()
		return
	}
	p.cmd.SysProcAttr = &syscall.SysProcAttr{}
	set_deathsig(p.cmd.SysProcAttr)
	p.setEnv()
	p.setDir()
	p.setLog()

	p.stdin, _ = p.cmd.StdinPipe()
	p.startTime = time.Now()
	p.state = STARTING
	err = p.cmd.Start()
	if err != nil {
		log.WithFields(log.Fields{"program": p.config.GetProgramName()}).Error("fail to start program")
		p.state = FATAL
		p.stopTime = time.Now()
		p.lock.Unlock()
		runCond.Signal()
	} else {
		if p.stdoutLog != nil {
			p.stdoutLog.SetPid(p.GetPid())
		}
		if p.stderrLog != nil {
			p.stderrLog.SetPid(p.GetPid())
		}
		log.WithFields(log.Fields{"program": p.config.GetProgramName()}).Info("success to start program")
		startSecs := p.config.GetInt("startsecs", 1)
		//Set startsec to 0 to indicate that the program needn't stay
		//running for any particular amount of time.
		if startSecs > 0 {
			p.state = RUNNING
		}
		p.lock.Unlock()
		log.WithFields(log.Fields{"program": p.config.GetProgramName()}).Debug("wait program exit")
		runCond.Signal()
		p.cmd.Wait()
		p.lock.Lock()
		p.stopTime = time.Now()
		if p.stopTime.Unix()-p.startTime.Unix() < int64(startSecs) {
			p.state = BACKOFF
		} else {
			p.state = EXITED
		}
		p.lock.Unlock()
	}

}

func (p *Process) Signal(sig os.Signal) error {
	p.lock.Lock()
	defer p.lock.Unlock()

	if p.cmd != nil && p.cmd.Process != nil {
		return p.cmd.Process.Signal(sig)
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
	dir := p.config.GetString("directory", "")
	if dir != "" {
		p.cmd.Dir = dir
		fmt.Printf("Directory has been set to: %s\n", dir)
	}
}

func (p *Process) setLog() {
	if p.config.IsProgram() {
		p.stdoutLog = p.createLogger(p.config.GetString("stdout_logfile", ""),
			int64(p.config.GetBytes("stdout_logfile_maxbytes", 50*1024*1024)),
			p.config.GetInt("stdout_logfile_backups", 10))
		capture_bytes := p.config.GetBytes("stdout_capture_maxbytes", 0)
		if capture_bytes > 0 && p.config.GetBool("stdout_events_enabled", false) {
			log.WithFields(log.Fields{"program": p.config.GetProgramName()}).Info("capture stdout process communication")
			p.stdoutLog = NewLogCaptureLogger(p.stdoutLog,
				capture_bytes,
				"PROCESS_COMMUNICATION_STDOUT",
				p.GetName(),
				p.GetGroup())
		}

		p.cmd.Stdout = p.stdoutLog

		p.stderrLog = p.createLogger(p.config.GetString("stderr_logfile", ""),
			int64(p.config.GetBytes("stderr_logfile_maxbytes", 50*1024*1024)),
			p.config.GetInt("stderr_logfile_backups", 10))

		capture_bytes = p.config.GetBytes("stderr_capture_maxbytes", 0)
		if capture_bytes > 0 && p.config.GetBool("stderr_events_enabled", false) {
			log.WithFields(log.Fields{"program": p.config.GetProgramName()}).Info("capture stderr process communication")
			p.stderrLog = NewLogCaptureLogger(p.stdoutLog,
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

func (p *Process) registerEventListener(eventListenerName string,
	events []string,
	stdin io.Reader,
	stdout io.Writer) {
	eventListener := NewEventListener(eventListenerName,
		p.supervisor_id,
		stdin,
		stdout,
		p.config.GetInt("buffer_size", 100))
	eventListenerManager.registerEventListener(eventListenerName, events, eventListener)
}

func (p *Process) unregisterEventListener(eventListenerName string) {
	eventListenerManager.unregisterEventListener(eventListenerName)
}

func (p *Process) createLogger(logFile string, maxBytes int64, backups int) Logger {
	var logger Logger
	logger = NewNullLogger()

	if logFile == "/dev/stdout" {
		logger = NewStdoutLogger()
	} else if logFile == "/dev/stderr" {
		logger = NewStderrLogger()
	} else if len(logFile) > 0 {
		logger = NewFileLogger(logFile, maxBytes, backups, NewNullLocker())
	}
	return logger
}

func (p *Process) setUser() error {
	userName := p.config.GetString("user", "")
	if len(userName) == 0 {
		return nil
	}
	u, err := user.Lookup(userName)
	if err != nil {
		return err
	}
	p.cmd.SysProcAttr = &syscall.SysProcAttr{}
	uid, err := strconv.ParseUint(u.Uid, 10, 32)
	if err != nil {
		return err
	}
	gid, err := strconv.ParseUint(u.Gid, 10, 32)
	if err != nil {
		return err
	}
	p.cmd.SysProcAttr.Credential = &syscall.Credential{Uid: uint32(uid), Gid: uint32(gid)}
	return nil
}

//convert a signal name to signal
func toSignal(signalName string) os.Signal {
	if signalName == "HUP" {
		return syscall.SIGHUP
	} else if signalName == "INT" {
		return syscall.SIGINT
	} else if signalName == "QUIT" {
		return syscall.SIGQUIT
	} else if signalName == "KILL" {
		return syscall.SIGKILL
	} else if signalName == "USR1" {
		return syscall.SIGUSR1
	} else if signalName == "USR2" {
		return syscall.SIGUSR2
	} else {
		return syscall.SIGTERM

	}

}

//send signal to process to stop it
func (p *Process) Stop(wait bool) {
	p.lock.Lock()
	defer p.lock.Unlock()
	p.stopByUser = true
	if p.cmd != nil && p.cmd.Process != nil {
		log.WithFields(log.Fields{"program": p.GetName()}).Info("stop the program")
		p.cmd.Process.Signal(toSignal(p.config.GetString("stopsignal", "")))
		if wait {
			p.cmd.Process.Wait()
		}
	}
}

func (p *Process) GetStatus() string {
	if p.cmd.ProcessState.Exited() {
		return p.cmd.ProcessState.String()
	}
	return "running"
}

func (p ProcessSortByPriority) Len() int {
	return len(p)
}

func (p ProcessSortByPriority) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}

func (p ProcessSortByPriority) Less(i, j int) bool {
	return p[i].GetPriority() < p[j].GetPriority()
}
