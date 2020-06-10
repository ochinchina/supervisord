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

	"github.com/mitchellh/go-homedir"
	"github.com/ochinchina/filechangemonitor"
	"github.com/r3labs/diff"
	"github.com/robfig/cron/v3"
	"github.com/stuartcarnie/gopm/config"
	"github.com/stuartcarnie/gopm/logger"
	"github.com/stuartcarnie/gopm/signals"
	"go.uber.org/zap"
)

var gShellArgs []string

func SetShellArgs(s []string) {
	gShellArgs = s
}

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

// String convert State to human readable string
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
	cmd          *exec.Cmd
	log          *zap.Logger
	startTime    time.Time
	stopTime     time.Time
	state        State
	cronID       cron.EntryID
	// true if process is starting
	inStart bool
	// true if the process is stopped by user
	stopByUser bool
	retryTimes *int32
	mu         sync.RWMutex
	stdin      io.WriteCloser
	StdoutLog  logger.Logger
	StderrLog  logger.Logger
	cfgMu      sync.RWMutex // protects config access
	config     *config.Process
}

// NewProcess create a new Process
func NewProcess(supervisorID string, cfg *config.Process) *Process {
	proc := &Process{
		supervisorID: supervisorID,
		config:       cfg,
		log:          zap.L().With(zap.String("program", cfg.Name)),
		state:        Stopped,
		retryTimes:   new(int32),
	}
	proc.addToCron()
	return proc
}

func (p *Process) UpdateConfig(config *config.Process) {
	changes, _ := diff.Diff(p.Config(), config)
	_ = changes
	p.cfgMu.Lock()
	p.config = config
	p.cfgMu.Unlock()
}

func (p *Process) Config() *config.Process {
	p.cfgMu.RLock()
	defer p.cfgMu.RUnlock()
	return p.config
}

// add this process to crontab
func (p *Process) addToCron() {
	cfg := p.config
	schedule := cfg.CronSchedule()
	if schedule == nil {
		return
	}

	p.log.Info("Scheduling program with cron", zap.String("cron", cfg.Cron))
	id := scheduler.Schedule(schedule, cron.FuncJob(func() {
		p.log.Debug("Running scheduled program")
		if !p.isRunning() {
			p.Start(false)
		}
	}))

	p.cronID = id
}

func (p *Process) removeFromCron() {
	s := p.Config().Cron
	if len(s) == 0 {
		return
	}

	p.log.Info("Removing program from cron schedule")
	scheduler.Remove(p.cronID)
	p.cronID = 0
}

// Start start the process
// Args:
//  wait - true, wait the program started or failed
func (p *Process) Start(wait bool) {
	p.log.Info("Starting program")
	p.mu.Lock()
	if p.inStart {
		p.log.Info("Program already starting")
		p.mu.Unlock()
		return
	}

	p.inStart = true
	p.stopByUser = false
	p.mu.Unlock()

	signalWaiter := func() {}
	waitCh := make(chan struct{})
	if wait {
		var once sync.Once
		signalWaiter = func() {
			once.Do(func() {
				close(waitCh)
			})
		}
	} else {
		close(waitCh)
	}

	go func() {
		defer func() {
			p.mu.Lock()
			p.inStart = false
			p.mu.Unlock()
			signalWaiter()
		}()

		for {
			p.run(signalWaiter)

			// avoid print too many logs if fail to start program too quickly
			if time.Now().Unix()-p.startTime.Unix() < 2 {
				time.Sleep(5 * time.Second)
			}

			if p.stopByUser {
				p.log.Info("Stopped by user, don't start it again")
				return
			}
			if !p.isAutoRestart() {
				p.log.Info("Auto restart disabled; won't restart")
				return
			}
		}
	}()

	<-waitCh
}

// Destroy stops the process and removes it from cron
func (p *Process) Destroy() {
	p.removeFromCron()
	p.Stop(false)
}

// Name returns the name of program
func (p *Process) Name() string {
	return p.Config().Name
}

// Group which group the program belongs to
func (p *Process) Group() string {
	return p.Config().Group
}

// Description get the process status description
func (p *Process) Description() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
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
		return p.stopTime.String()
	}
	return ""
}

// GetExitStatus get the exit status of the process if the program exit
func (p *Process) ExitStatus() int {
	p.mu.RLock()
	defer p.mu.RUnlock()

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

// GetPid get the pid of running process or 0 it is not in running status
func (p *Process) Pid() int {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.state == Stopped || p.state == Fatal || p.state == Unknown || p.state == Exited || p.state == Backoff {
		return 0
	}
	return p.cmd.Process.Pid
}

// GetState Get the process state
func (p *Process) State() State {
	return p.state
}

// GetStartTime get the process start time
func (p *Process) StartTime() time.Time {
	return p.startTime
}

// GetStopTime get the process stop time
func (p *Process) StopTime() time.Time {
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

// GetStdoutLogfile get the program stdout log file
func (p *Process) StdoutLogfile() string {
	fileName := p.Config().StdoutLogFile
	expandFile, err := homedir.Expand(fileName)
	if err != nil {
		return fileName
	}
	return expandFile
}

// GetStderrLogfile get the program stderr log file
func (p *Process) StderrLogfile() string {
	fileName := p.Config().StderrLogFile
	expandFile, err := homedir.Expand(fileName)
	if err != nil {
		return fileName
	}
	return expandFile
}

// SendProcessStdin send data to process stdin
func (p *Process) SendProcessStdin(chars string) error {
	if p.stdin != nil {
		_, err := p.stdin.Write([]byte(chars))
		return err
	}
	return fmt.Errorf("NO_FILE")
}

// check if the process should be
func (p *Process) isAutoRestart() bool {
	switch p.Config().AutoRestart {
	case config.AutoStartModeDefault:
		if p.cmd != nil && p.cmd.ProcessState != nil {
			exitCode, err := p.getExitCode()
			// If unexpected, the process will be restarted when the program exits
			// with an exit code that is not one of the exit codes associated with
			// this process’ configuration (see exitcodes).
			return err == nil && !p.inExitCodes(exitCode)
		}
		return false

	case config.AutoStartModeAlways:
		return true
	}
	return false
}

func (p *Process) inExitCodes(exitCode int) bool {
	for _, code := range p.Config().ExitCodes {
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
	return p.cmd.ProcessState.ExitCode(), nil
}

// check if the process is running or not
//
func (p *Process) isRunning() bool {
	if p.cmd != nil && p.cmd.Process != nil {
		if runtime.GOOS == "windows" {
			proc, err := os.FindProcess(p.cmd.Process.Pid)
			return proc != nil && err == nil
		}
		return p.cmd.Process.Signal(syscall.Signal(0)) == nil
	}
	return false
}

// create Command object for the program
func (p *Process) createProgramCommand() error {
	cfg := p.Config()

	args := strings.SplitN(cfg.Command, " ", 2)
	p.cmd = exec.Command(gShellArgs[0], append(gShellArgs[1:], cfg.Command)...)
	p.cmd.SysProcAttr = &syscall.SysProcAttr{}
	if p.setUser() != nil {
		p.log.Error("Failed to run as user", zap.String("user", cfg.User))
		return fmt.Errorf("failed to set user")
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
	cfg := p.Config()

	if cfg.RestartWhenBinaryChanged {
		absPath, err := filepath.Abs(programPath)
		if err != nil {
			absPath = programPath
		}
		AddProgramChangeMonitor(absPath, func(path string, mode filechangemonitor.FileChangeMode) {
			p.log.Info("Program binary changed")
			p.Stop(true)
			p.Start(true)
		})
	}
	dirMonitor := cfg.RestartDirectoryMonitor
	filePattern := cfg.RestartFilePattern
	if dirMonitor != "" {
		absDir, err := filepath.Abs(dirMonitor)
		if err != nil {
			absDir = dirMonitor
		}
		AddConfigChangeMonitor(absDir, filePattern, func(path string, mode filechangemonitor.FileChangeMode) {
			// fmt.Printf( "filePattern=%s, base=%s\n", filePattern, filepath.Base( path ) )
			// if matched, err := filepath.Match( filePattern, filepath.Base( path ) ); matched && err == nil {
			p.log.Info("Watched file for program is changed")
			p.Stop(true)
			p.Start(true)
			//}
		})
	}
}

// wait for the started program exit
func (p *Process) waitForExit() {
	p.cmd.Wait()
	if p.cmd.ProcessState != nil {
		p.log.Info("Program stopped", zap.Stringer("status", p.cmd.ProcessState))
	} else {
		p.log.Info("Program stopped")
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.stopTime = time.Now().Round(time.Millisecond)
	p.StdoutLog.Close()
	p.StderrLog.Close()
}

// fail to start the program
func (p *Process) failToStartProgram(reason string, finishedFn func()) {
	p.log.Error(reason)
	p.changeStateTo(Fatal)
	finishedFn()
}

// monitor if the program is in running before endTime
//
func (p *Process) monitorProgramIsRunning(endTime time.Time, monitorExited, programExited *int32) {
	// if time is not expired
	for time.Now().Before(endTime) && atomic.LoadInt32(programExited) == 0 {
		time.Sleep(time.Duration(100) * time.Millisecond)
	}
	atomic.StoreInt32(monitorExited, 1)

	p.mu.Lock()
	defer p.mu.Unlock()
	// if the program does not exit
	if atomic.LoadInt32(programExited) == 0 && p.state == Starting {
		p.log.Info("Successfully started program")
		p.changeStateTo(Running)
	}
}

func (p *Process) run(finishedFn func()) {
	p.mu.Lock()
	defer p.mu.Unlock()

	cfg := p.Config()

	// check if the program is in running state
	if p.isRunning() {
		p.log.Info("Program already running")
		finishedFn()
		return

	}
	p.startTime = time.Now().Round(time.Millisecond)
	atomic.StoreInt32(p.retryTimes, 0)
	startSecs := cfg.StartSeconds
	restartPause := cfg.RestartPause

	var once sync.Once
	finishedOnceFn := func() {
		once.Do(finishedFn)
	}

	// process is not expired and not stopped by user
	for !p.stopByUser {
		if restartPause > 0 && atomic.LoadInt32(p.retryTimes) != 0 {
			// pause
			p.mu.Unlock()
			p.log.Info("Delay program restart", zap.Duration("restart_pause_seconds", time.Duration(restartPause)))
			time.Sleep(time.Duration(restartPause))
			p.mu.Lock()
		}
		endTime := time.Now().Add(time.Duration(startSecs))
		p.changeStateTo(Starting)
		atomic.AddInt32(p.retryTimes, 1)

		err := p.createProgramCommand()
		if err != nil {
			p.failToStartProgram("Failed to create program", finishedOnceFn)
			break
		}

		err = p.cmd.Start()

		if err != nil {
			if atomic.LoadInt32(p.retryTimes) >= int32(cfg.StartRetries) {
				p.failToStartProgram(fmt.Sprintf("fail to start program with error:%v", err), finishedOnceFn)
				break
			} else {
				p.log.Error("Failed to start program", zap.Error(err))
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

		monitorExited := int32(0)
		programExited := int32(0)
		// Set startsec to 0 to indicate that the program needn't stay
		// running for any particular amount of time.
		if startSecs <= 0 {
			p.log.Info("Program started")
			p.changeStateTo(Running)
			go finishedOnceFn()
		} else {
			go func() {
				p.monitorProgramIsRunning(endTime, &monitorExited, &programExited)
				finishedOnceFn()
			}()
		}
		p.log.Debug("Waiting for program to exit")
		p.mu.Unlock()
		p.waitForExit()

		atomic.StoreInt32(&programExited, 1)
		// wait for monitor thread exit
		for atomic.LoadInt32(&monitorExited) == 0 {
			time.Sleep(time.Duration(10) * time.Millisecond)
		}

		p.mu.Lock()

		// if the program still in running after startSecs
		if p.state == Running {
			p.changeStateTo(Exited)
			p.log.Info("Program exited")
			break
		} else {
			p.changeStateTo(Backoff)
		}

		// The number of serial failure attempts that gopm will allow when attempting to
		// start the program before giving up and putting the process into an Fatal state
		// first start time is not the retry time
		if atomic.LoadInt32(p.retryTimes) >= int32(cfg.StartRetries) {
			p.failToStartProgram(fmt.Sprintf("Unable to run program; exceeded retry count: %d", cfg.StartRetries), finishedOnceFn)
			break
		}
	}
}

func (p *Process) changeStateTo(procState State) {
	p.state = procState
}

// Signal send signal to the process
//
// Args:
//   sig - the signal to the process
//   sigChildren - true: send the signal to the process and its children proess
//
func (p *Process) Signal(sig os.Signal, sigChildren bool) error {
	p.mu.RLock()
	defer p.mu.RUnlock()

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
	return fmt.Errorf("process not started")
}

func (p *Process) setEnv() {
	var env []string
	for k, v := range p.Config().Environment {
		env = append(env, k+"="+v)
	}

	if len(env) != 0 {
		p.cmd.Env = append(os.Environ(), env...)
	} else {
		p.cmd.Env = os.Environ()
	}
}

func (p *Process) setDir() {
	dir := p.Config().Directory
	if dir != "" {
		p.cmd.Dir = dir
	}
}

func (p *Process) setLog() {
	cfg := p.Config()

	p.StdoutLog = p.createLogger(p.StdoutLogfile(), int64(cfg.StdoutLogFileMaxBytes), cfg.StdoutLogfileBackups)
	p.cmd.Stdout = p.StdoutLog

	if cfg.RedirectStderr {
		p.StderrLog = p.StdoutLog
	} else {
		p.StderrLog = p.createLogger(p.StderrLogfile(), int64(cfg.StderrLogFileMaxBytes), cfg.StderrLogfileBackups)
	}

	p.cmd.Stderr = p.StderrLog
}

func (p *Process) createLogger(logFile string, maxBytes int64, backups int) logger.Logger {
	return logger.NewLogger(p.Name(), logFile, logger.NewNullLocker(), maxBytes, backups)
}

func (p *Process) setUser() error {
	userName := p.Config().User
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
	return nil
}

// Stop send signal to process to stop it
func (p *Process) Stop(wait bool) {
	p.mu.Lock()
	p.stopByUser = true
	isRunning := p.isRunning()
	p.mu.Unlock()
	if !isRunning {
		return
	}
	p.log.Info("Stopping program")

	cfg := p.Config()

	sigs := cfg.StopSignals
	if len(sigs) == 0 {
		p.log.Error("Missing signals; defaulting to KILL")
		sigs = []string{"KILL"}
	}

	waitDur := cfg.StopWaitSeconds
	stopAsGroup := cfg.StopAsGroup
	killAsGroup := cfg.KillAsGroup
	if stopAsGroup && !killAsGroup {
		p.log.Error("Invalid group configuration; stop_as_group=true and kill_as_group=false")
		killAsGroup = stopAsGroup
	}

	ch := make(chan struct{})
	go func() {
		defer close(ch)

		for i := 0; i < len(sigs); i++ {
			// send signal to process
			sig, err := signals.ToSignal(sigs[i])
			if err != nil {
				continue
			}

			p.log.Info("Send stop signal to program", zap.String("signal", sigs[i]))
			_ = p.Signal(sig, stopAsGroup)

			endTime := time.Now().Add(waitDur)
			for endTime.After(time.Now()) {
				// if it already exits
				if p.state != Starting && p.state != Running && p.state != Stopping {
					p.log.Info("Program exited")
					return
				}
				time.Sleep(10 * time.Millisecond)
			}
		}

		p.log.Info("Program did not stop in time, sending KILL")
		p.Signal(syscall.SIGKILL, killAsGroup)
	}()

	if wait {
		<-ch
	}
}

// GetStatus get the status of program in string
func (p *Process) GetStatus() string {
	if p.cmd.ProcessState.Exited() {
		return p.cmd.ProcessState.String()
	}
	return "running"
}
