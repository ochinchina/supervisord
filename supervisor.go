package main

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/ochinchina/supervisord/config"
	"github.com/ochinchina/supervisord/events"
	"github.com/ochinchina/supervisord/faults"
	"github.com/ochinchina/supervisord/logger"
	"github.com/ochinchina/supervisord/process"
	"github.com/ochinchina/supervisord/signals"
	"github.com/ochinchina/supervisord/types"
	"github.com/ochinchina/supervisord/util"

	log "github.com/sirupsen/logrus"
)

const (
	// SupervisorVersion the version of supervisor
	SupervisorVersion = "3.0"
)

// Supervisor manage all the processes defined in the supervisor configuration file.
// All the supervisor public interface is defined in this class
type Supervisor struct {
	config     *config.Config   // supervisor configuration
	procMgr    *process.Manager // process manager
	xmlRPC     *XMLRPC          // XMLRPC interface
	logger     logger.Logger    // logger manager
	lock       sync.Mutex
	restarting bool // if supervisor is in restarting state
}

// StartProcessArgs arguments for starting a process
type StartProcessArgs struct {
	Name string // program name
	Wait bool   `default:"true"` // Wait the program starting finished
}

// ProcessStdin  process stdin from client
type ProcessStdin struct {
	Name  string // program name
	Chars string // inputs from client
}

// RemoteCommEvent remove communication event from client side
type RemoteCommEvent struct {
	Type string // the event type
	Data string // the data of event
}

// StateInfo describe the state of supervisor
type StateInfo struct {
	Statecode int    `xml:"statecode"`
	Statename string `xml:"statename"`
}

// RPCTaskResult result of some remote commands
type RPCTaskResult struct {
	Name        string `xml:"name"`        // the program name
	Group       string `xml:"group"`       // the group of the program
	Status      int    `xml:"status"`      // the status of the program
	Description string `xml:"description"` // the description of program
}

// LogReadInfo the input argument to read the log of supervisor
type LogReadInfo struct {
	Offset int // the log offset
	Length int // the length of log to read
}

// ProcessLogReadInfo the input argument to read the log of program
type ProcessLogReadInfo struct {
	Name   string // the program name
	Offset int    // the offset of the program log
	Length int    // the length of log to read
}

// ProcessTailLog the output of tail the program log
type ProcessTailLog struct {
	LogData  string
	Offset   int64
	Overflow bool
}

// NewSupervisor create a Supervisor object with supervisor configuration file
func NewSupervisor(configFile string) *Supervisor {
	return &Supervisor{config: config.NewConfig(configFile),
		procMgr:    process.NewManager(),
		xmlRPC:     NewXMLRPC(),
		restarting: false}
}

// GetConfig get the loaded supervisor configuration
func (s *Supervisor) GetConfig() *config.Config {
	return s.config
}

// GetVersion get the version of supervisor
func (s *Supervisor) GetVersion(r *http.Request, args *struct{}, reply *struct{ Version string }) error {
	reply.Version = SupervisorVersion
	return nil
}

// GetSupervisorVersion get the supervisor version
func (s *Supervisor) GetSupervisorVersion(r *http.Request, args *struct{}, reply *struct{ Version string }) error {
	reply.Version = SupervisorVersion
	return nil
}

// GetIdentification get the supervisor identifier configured in the file
func (s *Supervisor) GetIdentification(r *http.Request, args *struct{}, reply *struct{ ID string }) error {
	reply.ID = s.GetSupervisorID()
	return nil
}

// GetSupervisorID get the supervisor identifier from configuration file
func (s *Supervisor) GetSupervisorID() string {
	entry, ok := s.config.GetSupervisord()
	if !ok {
		return "supervisor"
	}
	return entry.GetString("identifier", "supervisor")
}

// GetState get the state of supervisor
func (s *Supervisor) GetState(r *http.Request, args *struct{}, reply *struct{ StateInfo StateInfo }) error {
	// statecode    statename
	// =======================
	// 2            FATAL
	// 1            RUNNING
	// 0            RESTARTING
	// -1           SHUTDOWN
	log.Debug("Get state")
	reply.StateInfo.Statecode = 1
	reply.StateInfo.Statename = "RUNNING"
	return nil
}

// GetPrograms Get all the name of programs
//
// Return the name of all the programs
func (s *Supervisor) GetPrograms() []string {
	return s.config.GetProgramNames()
}

// GetPID get the pid of supervisor
func (s *Supervisor) GetPID(r *http.Request, args *struct{}, reply *struct{ Pid int }) error {
	reply.Pid = os.Getpid()
	return nil
}

// ReadLog read the log of supervisor
func (s *Supervisor) ReadLog(r *http.Request, args *LogReadInfo, reply *struct{ Log string }) error {
	data, err := s.logger.ReadLog(int64(args.Offset), int64(args.Length))
	reply.Log = data
	return err
}

// ClearLog clear the supervisor log
func (s *Supervisor) ClearLog(r *http.Request, args *struct{}, reply *struct{ Ret bool }) error {
	err := s.logger.ClearAllLogFile()
	reply.Ret = err == nil
	return err
}

// Shutdown the supervisor
func (s *Supervisor) Shutdown(r *http.Request, args *struct{}, reply *struct{ Ret bool }) error {
	reply.Ret = true
	log.Info("received rpc request to stop all processes & exit")
	s.procMgr.StopAllProcesses()
	go func() {
		time.Sleep(1 * time.Second)
		os.Exit(0)
	}()
	return nil
}

// Restart the supervisor
func (s *Supervisor) Restart(r *http.Request, args *struct{}, reply *struct{ Ret bool }) error {
	log.Info("Receive instruction to restart")
	s.restarting = true
	reply.Ret = true
	return nil
}

// IsRestarting check if supervisor is in restarting state
func (s *Supervisor) IsRestarting() bool {
	return s.restarting
}

func getProcessInfo(proc *process.Process) *types.ProcessInfo {
	return &types.ProcessInfo{Name: proc.GetName(),
		Group:         proc.GetGroup(),
		Description:   proc.GetDescription(),
		Start:         int(proc.GetStartTime().Unix()),
		Stop:          int(proc.GetStopTime().Unix()),
		Now:           int(time.Now().Unix()),
		State:         int(proc.GetState()),
		Statename:     proc.GetState().String(),
		Spawnerr:      "",
		Exitstatus:    proc.GetExitstatus(),
		Logfile:       proc.GetStdoutLogfile(),
		StdoutLogfile: proc.GetStdoutLogfile(),
		StderrLogfile: proc.GetStderrLogfile(),
		Pid:           proc.GetPid()}

}

// GetAllProcessInfo get all the program information managed by supervisor
func (s *Supervisor) GetAllProcessInfo(r *http.Request, args *struct{}, reply *struct{ AllProcessInfo []types.ProcessInfo }) error {
	reply.AllProcessInfo = make([]types.ProcessInfo, 0)
	s.procMgr.ForEachProcess(func(proc *process.Process) {
		procInfo := getProcessInfo(proc)
		reply.AllProcessInfo = append(reply.AllProcessInfo, *procInfo)
	})
	types.SortProcessInfos(reply.AllProcessInfo)
	return nil
}

// GetProcessInfo get the process information of one program
func (s *Supervisor) GetProcessInfo(r *http.Request, args *struct{ Name string }, reply *struct{ ProcInfo types.ProcessInfo }) error {
	log.Info("Get process info of: ", args.Name)
	proc := s.procMgr.Find(args.Name)
	if proc == nil {
		return fmt.Errorf("BAD_NAME no process named %s", args.Name)
	}

	reply.ProcInfo = *getProcessInfo(proc)
	return nil
}

// StartProcess start the given program
func (s *Supervisor) StartProcess(r *http.Request, args *StartProcessArgs, reply *struct{ Success bool }) error {
	procs := s.procMgr.FindMatch(args.Name)

	if len(procs) <= 0 {
		return fmt.Errorf("fail to find process %s", args.Name)
	}
	for _, proc := range procs {
		proc.Start(args.Wait)
	}
	reply.Success = true
	return nil
}

// StartAllProcesses start all the programs
func (s *Supervisor) StartAllProcesses(r *http.Request, args *struct {
	Wait bool `default:"true"`
}, reply *struct{ RPCTaskResults []RPCTaskResult }) error {

	finishedProcCh := make(chan *process.Process)

	n := s.procMgr.AsyncForEachProcess(func(proc *process.Process) {
		proc.Start(args.Wait)
	}, finishedProcCh)

	for i := 0; i < n; i++ {
		proc, ok := <-finishedProcCh
		if ok {
			processInfo := *getProcessInfo(proc)
			reply.RPCTaskResults = append(reply.RPCTaskResults, RPCTaskResult{
				Name:        processInfo.Name,
				Group:       processInfo.Group,
				Status:      faults.Success,
				Description: "OK",
			})
		}
	}
	return nil
}

// StartProcessGroup start all the processes in one group
func (s *Supervisor) StartProcessGroup(r *http.Request, args *StartProcessArgs, reply *struct{ AllProcessInfo []types.ProcessInfo }) error {
	log.WithFields(log.Fields{"group": args.Name}).Info("start process group")
	finishedProcCh := make(chan *process.Process)

	n := s.procMgr.AsyncForEachProcess(func(proc *process.Process) {
		if proc.GetGroup() == args.Name {
			proc.Start(args.Wait)
		}
	}, finishedProcCh)

	for i := 0; i < n; i++ {
		proc, ok := <-finishedProcCh
		if ok && proc.GetGroup() == args.Name {
			reply.AllProcessInfo = append(reply.AllProcessInfo, *getProcessInfo(proc))
		}
	}

	return nil
}

// StopProcess stop given program
func (s *Supervisor) StopProcess(r *http.Request, args *StartProcessArgs, reply *struct{ Success bool }) error {
	log.WithFields(log.Fields{"program": args.Name}).Info("stop process")
	procs := s.procMgr.FindMatch(args.Name)
	if len(procs) <= 0 {
		return fmt.Errorf("fail to find process %s", args.Name)
	}
	for _, proc := range procs {
		proc.Stop(args.Wait)
	}
	reply.Success = true
	return nil
}

// StopProcessGroup stop all processes in one group
func (s *Supervisor) StopProcessGroup(r *http.Request, args *StartProcessArgs, reply *struct{ AllProcessInfo []types.ProcessInfo }) error {
	log.WithFields(log.Fields{"group": args.Name}).Info("stop process group")
	finishedProcCh := make(chan *process.Process)
	n := s.procMgr.AsyncForEachProcess(func(proc *process.Process) {
		if proc.GetGroup() == args.Name {
			proc.Stop(args.Wait)
		}
	}, finishedProcCh)

	for i := 0; i < n; i++ {
		proc, ok := <-finishedProcCh
		if ok && proc.GetGroup() == args.Name {
			reply.AllProcessInfo = append(reply.AllProcessInfo, *getProcessInfo(proc))
		}
	}
	return nil
}

// StopAllProcesses stop all programs managed by supervisor
func (s *Supervisor) StopAllProcesses(r *http.Request, args *struct {
	Wait bool `default:"true"`
}, reply *struct{ RPCTaskResults []RPCTaskResult }) error {
	finishedProcCh := make(chan *process.Process)

	n := s.procMgr.AsyncForEachProcess(func(proc *process.Process) {
		proc.Stop(args.Wait)
	}, finishedProcCh)

	for i := 0; i < n; i++ {
		proc, ok := <-finishedProcCh
		if ok {
			processInfo := *getProcessInfo(proc)
			reply.RPCTaskResults = append(reply.RPCTaskResults, RPCTaskResult{
				Name:        processInfo.Name,
				Group:       processInfo.Group,
				Status:      faults.Success,
				Description: "OK",
			})
		}
	}
	return nil
}

// SignalProcess send a signal to running program
func (s *Supervisor) SignalProcess(r *http.Request, args *types.ProcessSignal, reply *struct{ Success bool }) error {
	procs := s.procMgr.FindMatch(args.Name)
	if len(procs) <= 0 {
		reply.Success = false
		return fmt.Errorf("No process named %s", args.Name)
	}
	sig, err := signals.ToSignal(args.Signal)
	if err == nil {
		for _, proc := range procs {
			proc.Signal(sig, false)
		}
	}
	reply.Success = true
	return nil
}

// SignalProcessGroup send signal to all processes in one group
func (s *Supervisor) SignalProcessGroup(r *http.Request, args *types.ProcessSignal, reply *struct{ AllProcessInfo []types.ProcessInfo }) error {
	s.procMgr.ForEachProcess(func(proc *process.Process) {
		if proc.GetGroup() == args.Name {
			sig, err := signals.ToSignal(args.Signal)
			if err == nil {
				proc.Signal(sig, false)
			}
		}
	})

	s.procMgr.ForEachProcess(func(proc *process.Process) {
		if proc.GetGroup() == args.Name {
			reply.AllProcessInfo = append(reply.AllProcessInfo, *getProcessInfo(proc))
		}
	})
	return nil
}

// SignalAllProcesses send signal to all the processes in the supervisor
func (s *Supervisor) SignalAllProcesses(r *http.Request, args *types.ProcessSignal, reply *struct{ AllProcessInfo []types.ProcessInfo }) error {
	s.procMgr.ForEachProcess(func(proc *process.Process) {
		sig, err := signals.ToSignal(args.Signal)
		if err == nil {
			proc.Signal(sig, false)
		}
	})
	s.procMgr.ForEachProcess(func(proc *process.Process) {
		reply.AllProcessInfo = append(reply.AllProcessInfo, *getProcessInfo(proc))
	})
	return nil
}

// SendProcessStdin send data to program through stdin
func (s *Supervisor) SendProcessStdin(r *http.Request, args *ProcessStdin, reply *struct{ Success bool }) error {
	proc := s.procMgr.Find(args.Name)
	if proc == nil {
		log.WithFields(log.Fields{"program": args.Name}).Error("program does not exist")
		return fmt.Errorf("NOT_RUNNING")
	}
	if proc.GetState() != process.Running {
		log.WithFields(log.Fields{"program": args.Name}).Error("program does not run")
		return fmt.Errorf("NOT_RUNNING")
	}
	err := proc.SendProcessStdin(args.Chars)
	if err == nil {
		reply.Success = true
	} else {
		reply.Success = false
	}
	return err
}

// SendRemoteCommEvent emit a remote communication event
func (s *Supervisor) SendRemoteCommEvent(r *http.Request, args *RemoteCommEvent, reply *struct{ Success bool }) error {
	events.EmitEvent(events.NewRemoteCommunicationEvent(args.Type, args.Data))
	reply.Success = true
	return nil
}

// Reload supervisord configuration.
func (s *Supervisor) Reload(restart bool) (addedGroup []string, changedGroup []string, removedGroup []string, err error) {
	s.lock.Lock()
	defer s.lock.Unlock()
	// get the previous loaded programs
	prevPrograms := s.config.GetProgramNames()
	prevProgGroup := s.config.ProgramGroup.Clone()

	loadedPrograms, err := s.config.Load()

	if checkErr := s.checkRequiredResources(); checkErr != nil {
		log.Error(checkErr)
		os.Exit(1)

	}
	if err == nil {
		s.setSupervisordInfo()
		s.startEventListeners()
		s.createPrograms(prevPrograms)
		if restart {
			s.startHTTPServer()
		}
		s.startAutoStartPrograms()
	}
	removedPrograms := util.Sub(prevPrograms, loadedPrograms)
	for _, removedProg := range removedPrograms {
		log.WithFields(log.Fields{"program": removedProg}).Info("the program is removed and will be stopped")
		s.config.RemoveProgram(removedProg)
		proc := s.procMgr.Remove(removedProg)
		if proc != nil {
			proc.Stop(false)
		}

	}
	addedGroup, changedGroup, removedGroup = s.config.ProgramGroup.Sub(prevProgGroup)
	return addedGroup, changedGroup, removedGroup, err

}

// WaitForExit waits for supervisord to exit
func (s *Supervisor) WaitForExit() {
	for {
		if s.IsRestarting() {
			s.procMgr.StopAllProcesses()
			break
		}
		time.Sleep(10 * time.Second)
	}
}

func (s *Supervisor) createPrograms(prevPrograms []string) {

	programs := s.config.GetProgramNames()
	for _, entry := range s.config.GetPrograms() {
		s.procMgr.CreateProcess(s.GetSupervisorID(), entry)
	}
	removedPrograms := util.Sub(prevPrograms, programs)
	for _, p := range removedPrograms {
		s.procMgr.Remove(p)
	}
}

func (s *Supervisor) startAutoStartPrograms() {
	s.procMgr.StartAutoStartPrograms()
}

func (s *Supervisor) startEventListeners() {
	eventListeners := s.config.GetEventListeners()
	for _, entry := range eventListeners {
		proc := s.procMgr.CreateProcess(s.GetSupervisorID(), entry)
		proc.Start(false)
	}

	if len(eventListeners) > 0 {
		time.Sleep(1 * time.Second)
	}
}

func (s *Supervisor) startHTTPServer() {
	httpServerConfig, ok := s.config.GetInetHTTPServer()
	s.xmlRPC.Stop()
	if ok {
		addr := httpServerConfig.GetString("port", "")
		if addr != "" {
			cond := sync.NewCond(&sync.Mutex{})
			cond.L.Lock()
			defer cond.L.Unlock()
			go s.xmlRPC.StartInetHTTPServer(httpServerConfig.GetString("username", ""),
				httpServerConfig.GetString("password", ""),
				addr,
				s,
				func() {
					cond.L.Lock()
					cond.Signal()
					cond.L.Unlock()
				})
			cond.Wait()
		}
	}

	httpServerConfig, ok = s.config.GetUnixHTTPServer()
	if ok {
		env := config.NewStringExpression("here", s.config.GetConfigFileDir())
		sockFile, err := env.Eval(httpServerConfig.GetString("file", "/tmp/supervisord.sock"))
		if err == nil {
			cond := sync.NewCond(&sync.Mutex{})
			cond.L.Lock()
			defer cond.L.Unlock()
			go s.xmlRPC.StartUnixHTTPServer(httpServerConfig.GetString("username", ""),
				httpServerConfig.GetString("password", ""),
				sockFile,
				s,
				func() {
					cond.L.Lock()
					cond.Signal()
					cond.L.Unlock()
				})
			cond.Wait()
		}
	}

}

func (s *Supervisor) setSupervisordInfo() {
	supervisordConf, ok := s.config.GetSupervisord()
	if ok {
		// set supervisord log

		env := config.NewStringExpression("here", s.config.GetConfigFileDir())
		logFile, err := env.Eval(supervisordConf.GetString("logfile", "supervisord.log"))
		if err != nil {
			logFile, err = process.PathExpand(logFile)
		}
		if logFile == "/dev/stdout" {
			return
		}
		logEventEmitter := logger.NewNullLogEventEmitter()
		s.logger = logger.NewNullLogger(logEventEmitter)
		if err == nil {
			logfileMaxbytes := int64(supervisordConf.GetBytes("logfile_maxbytes", 50*1024*1024))
			logfileBackups := supervisordConf.GetInt("logfile_backups", 10)
			loglevel := supervisordConf.GetString("loglevel", "info")
			props := make(map[string]string)
			s.logger = logger.NewLogger("supervisord", logFile, &sync.Mutex{}, logfileMaxbytes, logfileBackups, props, logEventEmitter)
			log.SetLevel(toLogLevel(loglevel))
			log.SetFormatter(&log.TextFormatter{DisableColors: true, FullTimestamp: true})
			log.SetOutput(s.logger)
		}
		// set the pid
		pidfile, err := env.Eval(supervisordConf.GetString("pidfile", "supervisord.pid"))
		if err == nil {
			f, err := os.Create(pidfile)
			if err == nil {
				fmt.Fprintf(f, "%d", os.Getpid())
				f.Close()
			}
		}
	}
}

func toLogLevel(level string) log.Level {
	switch strings.ToLower(level) {
	case "critical":
		return log.FatalLevel
	case "error":
		return log.ErrorLevel
	case "warn":
		return log.WarnLevel
	case "info":
		return log.InfoLevel
	default:
		return log.DebugLevel
	}
}

// ReloadConfig reloads supervisord configuration file
func (s *Supervisor) ReloadConfig(r *http.Request, args *struct{}, reply *types.ReloadConfigResult) error {
	log.Info("start to reload config")
	addedGroup, changedGroup, removedGroup, err := s.Reload(false)
	if len(addedGroup) > 0 {
		log.WithFields(log.Fields{"groups": strings.Join(addedGroup, ",")}).Info("added groups")
	}

	if len(changedGroup) > 0 {
		log.WithFields(log.Fields{"groups": strings.Join(changedGroup, ",")}).Info("changed groups")
	}

	if len(removedGroup) > 0 {
		log.WithFields(log.Fields{"groups": strings.Join(removedGroup, ",")}).Info("removed groups")
	}
	reply.AddedGroup = addedGroup
	reply.ChangedGroup = changedGroup
	reply.RemovedGroup = removedGroup
	return err
}

// AddProcessGroup adds a process group to the supervisor
func (s *Supervisor) AddProcessGroup(r *http.Request, args *struct{ Name string }, reply *struct{ Success bool }) error {
	reply.Success = false
	return nil
}

// RemoveProcessGroup removes a process group from the supervisor
func (s *Supervisor) RemoveProcessGroup(r *http.Request, args *struct{ Name string }, reply *struct{ Success bool }) error {
	reply.Success = false
	return nil
}

// ReadProcessStdoutLog reads stdout of given program
func (s *Supervisor) ReadProcessStdoutLog(r *http.Request, args *ProcessLogReadInfo, reply *struct{ LogData string }) error {
	proc := s.procMgr.Find(args.Name)
	if proc == nil {
		return fmt.Errorf("No such process %s", args.Name)
	}
	var err error
	reply.LogData, err = proc.StdoutLog.ReadLog(int64(args.Offset), int64(args.Length))
	return err
}

// ReadProcessStderrLog reads stderr log of given program
func (s *Supervisor) ReadProcessStderrLog(r *http.Request, args *ProcessLogReadInfo, reply *struct{ LogData string }) error {
	proc := s.procMgr.Find(args.Name)
	if proc == nil {
		return fmt.Errorf("No such process %s", args.Name)
	}
	var err error
	reply.LogData, err = proc.StderrLog.ReadLog(int64(args.Offset), int64(args.Length))
	return err
}

// TailProcessStdoutLog tails stdout of the program
func (s *Supervisor) TailProcessStdoutLog(r *http.Request, args *ProcessLogReadInfo, reply *ProcessTailLog) error {
	proc := s.procMgr.Find(args.Name)
	if proc == nil {
		return fmt.Errorf("No such process %s", args.Name)
	}
	var err error
	reply.LogData, reply.Offset, reply.Overflow, err = proc.StdoutLog.ReadTailLog(int64(args.Offset), int64(args.Length))
	return err
}

// TailProcessStderrLog tails stderr of the program
func (s *Supervisor) TailProcessStderrLog(r *http.Request, args *ProcessLogReadInfo, reply *ProcessTailLog) error {
	proc := s.procMgr.Find(args.Name)
	if proc == nil {
		return fmt.Errorf("No such process %s", args.Name)
	}
	var err error
	reply.LogData, reply.Offset, reply.Overflow, err = proc.StderrLog.ReadTailLog(int64(args.Offset), int64(args.Length))
	return err
}

// ClearProcessLogs clears log of given program
func (s *Supervisor) ClearProcessLogs(r *http.Request, args *struct{ Name string }, reply *struct{ Success bool }) error {
	proc := s.procMgr.Find(args.Name)
	if proc == nil {
		return fmt.Errorf("No such process %s", args.Name)
	}
	err1 := proc.StdoutLog.ClearAllLogFile()
	err2 := proc.StderrLog.ClearAllLogFile()
	reply.Success = err1 == nil && err2 == nil
	if err1 != nil {
		return err1
	}
	return err2
}

// ClearAllProcessLogs clears logs of all programs
func (s *Supervisor) ClearAllProcessLogs(r *http.Request, args *struct{}, reply *struct{ RPCTaskResults []RPCTaskResult }) error {

	s.procMgr.ForEachProcess(func(proc *process.Process) {
		proc.StdoutLog.ClearAllLogFile()
		proc.StderrLog.ClearAllLogFile()
		procInfo := getProcessInfo(proc)
		reply.RPCTaskResults = append(reply.RPCTaskResults, RPCTaskResult{
			Name:        procInfo.Name,
			Group:       procInfo.Group,
			Status:      faults.Success,
			Description: "OK",
		})
	})

	return nil
}

// GetManager get the Manager object created by supervisor
func (s *Supervisor) GetManager() *process.Manager {
	return s.procMgr
}
