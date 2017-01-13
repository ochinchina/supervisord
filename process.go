package main

import (
	"fmt"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"strings"
	"strconv"
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

func (p ProcessState)String() string {
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
	config    *ConfigEntry
	cmd       *exec.Cmd
	startTime time.Time
	stopTime  time.Time
	state     ProcessState
	inStart	bool
	lock      sync.RWMutex
}

func NewProcess(config *ConfigEntry) *Process {
	proc := &Process{ config: config,
			cmd: nil,
			startTime: time.Unix(0,0),
			stopTime: time.Unix(0,0),
			state: STOPPED }
	proc.config = config
	proc.cmd = nil
	proc.state = STOPPED
	proc.inStart = false

	//start the process if autostart is set to true
	if( proc.isAutoStart() ) {
		proc.Start()
	}

	return proc
}

func (p *Process) Start() {
	p.lock.Lock()
	if p.inStart {
		p.lock.Unlock()
		return
	}

	p.inStart = true
	p.lock.Unlock()

	go func() {
		retryTimes := 0

		for {
			p.run()
			if (p.stopTime.Unix() - p.startTime.Unix()) < int64(p.getStartSeconds()) {
				retryTimes ++
			} else {
				retryTimes = 0
			}
			if retryTimes >= p.getStartRetries() || !p.isAutoRestart() {
				break
			}
		}
		p.lock.Lock()
		p.inStart = false
		p.lock.Unlock()
	}()
}

func (p *Process) GetName() string {
	return p.config.Name
}

func (p *Process) GetGroup() string {
	return p.config.Group
}

func (p *Process) GetDescription() string {
	if p.state == RUNNING {
		d := time.Now().Sub( p.startTime )
		return fmt.Sprintf( "pid %d, uptime %02d:%02d:%02d", p.cmd.Process.Pid, int(d.Hours()), int(d.Minutes()), int(d.Seconds() ) )
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
	return p.state;
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
		return time.Unix(0,0)
	default:
		return p.stopTime
	}
}

func (p *Process) GetStdoutLogfile() string {
	return p.config.GetString( "stdout_logfile", "/dev/null" )
}

func (p *Process) GetStderrLogfile() string {
	return p.config.GetString( "stderr_logfile", "/dev/null" )
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
	return p.config.GetInt( "priority", 999 )
}

func (p *Process) getNumberProcs() int {
	return p.config.GetInt( "numprocs", 1 )
}

func (p *Process) SendProcessStdin( chars string ) error {
	return fmt.Errorf( "NO_FILE")
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

func (p *Process) run() {
	p.lock.Lock()
	if p.cmd != nil && !p.cmd.ProcessState.Exited() {
		p.lock.Unlock()
		return
	}
	p.cmd = exec.Command("/bin/sh", "-c", p.config.GetString("command", ""))
	p.setEnv()
	p.setLog()

	p.startTime = time.Now()
	p.state = STARTING
	err := p.cmd.Start()
	if err != nil {
		log.WithFields( log.Fields{"program":p.config.GetProgramName()}).Error("fail to start program")
		p.state = FATAL
		p.stopTime = time.Now()
		p.lock.Unlock()
	} else {
		log.WithFields( log.Fields{"program":p.config.GetProgramName()}).Info("success to start program")
		startSecs := p.config.GetInt( "startsecs", 1 )
			//Set startsec to 0 to indicate that the program needn't stay 
			//running for any particular amount of time.
		if startSecs > 0 {
			p.state = RUNNING
		}
		p.lock.Unlock()
		p.cmd.Wait()
		p.lock.Lock()
		p.stopTime = time.Now()
		if p.stopTime.Unix() - p.startTime.Unix() < int64(startSecs) {
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
		return p.cmd.Process.Signal( sig )
	}

	return fmt.Errorf( "process is not started" )
}

func (p *Process) setEnv() {
	env := p.config.GetEnv("environment")
	if len(env) != 0 {
		p.cmd.Env = append(os.Environ(), env...)
	} else {
		p.cmd.Env = os.Environ()
	}
}

func (p *Process) setLog() {
	logFile := p.config.GetString("stdout_logfile", "")

	p.cmd.Stdout = NewNullLogger()

	if len(logFile) > 0 {
		p.cmd.Stdout = NewLogger( logFile,  
					int64(p.config.GetBytes( "stdout_logfile_maxbytes", 50*1024*1024 )), 
					p.config.GetInt( "stdout_logfile_backups", 10 ) )

	}

	logFile = p.config.GetString("stderr_logfile", "")
	p.cmd.Stderr = NewNullLogger()
	if len(logFile) > 0 {
		p.cmd.Stderr = NewLogger( logFile,
					int64( p.config.GetBytes( "stderr_logfile_maxbytes", 50*1024*1024 ) ),
					 p.config.GetInt( "stderr_logfile_backups", 10 ) )
	}
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
func (p *Process) Stop() {
	p.lock.Lock()
	defer p.lock.Unlock()
	if p.cmd != nil && p.cmd.Process != nil {
		p.cmd.Process.Signal(toSignal(p.config.GetString("stopsignal", "")))
	}
}

func (p *Process) GetStatus() string {
	if p.cmd.ProcessState.Exited() {
		return p.cmd.ProcessState.String()
	} else {
		return "running"
	}
}
