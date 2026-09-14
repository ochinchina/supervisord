package main

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ochinchina/supervisord/config"
	"github.com/ochinchina/supervisord/events"
	"github.com/ochinchina/supervisord/faults"
	"github.com/ochinchina/supervisord/logger"
	"github.com/ochinchina/supervisord/process"
	"github.com/ochinchina/supervisord/types"

	log "github.com/sirupsen/logrus"
)

const (
	// SupervisorVersion the version of supervisor
	SupervisorVersion = "3.0"
)

type ReloadAction int

const (
	RestartAll ReloadAction = iota
	RestartChanged
)

// Supervisor manage all the processes defined in the supervisor configuration file.
// All the supervisor public interface is defined in this class
type Supervisor struct {
	activeConfig  *config.Config   // supervisor configuration
	pendingConfig *config.Config   // pending configuration when reloading
	procMgr       *process.Manager // process manager
	xmlRPC        *XMLRPC          // XMLRPC interface
	logger        logger.Logger    // logger manager
	lock          sync.Mutex
	restarting    atomic.Bool // if supervisor is in restarting state
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
	Offset   int
	Overflow bool
}

// NewSupervisor create a Supervisor object with supervisor configuration file
func NewSupervisor(configFile string) *Supervisor {
	s := &Supervisor{activeConfig: config.NewConfig(configFile),
		pendingConfig: nil,
		procMgr:       process.NewManager(),
		xmlRPC:        NewXMLRPC()}

	s.setSupervisordInfo()

	return s
}

// GetConfig get the loaded supervisor configuration
func (s *Supervisor) GetConfig() *config.Config {
	return s.activeConfig
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
	entry, ok := s.activeConfig.GetSupervisord()
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
	if s.IsRestarting() {
		reply.StateInfo.Statecode = 0
		reply.StateInfo.Statename = "RESTARTING"
	} else {
		reply.StateInfo.Statecode = 1
		reply.StateInfo.Statename = "RUNNING"
	}
	return nil
}

// GetPrograms Get all the name of programs
//
// Return the name of all the programs
func (s *Supervisor) GetPrograms() []string {
	return s.activeConfig.GetProgramNames()
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
	s.restarting.Store(true)
	err := s.Reload(RestartAll)
	reply.Ret = err == nil
	s.restarting.Store(false)
	return nil
}

// IsRestarting check if supervisor is in restarting state
func (s *Supervisor) IsRestarting() bool {
	return s.restarting.Load()
}

func stateToFaults(state process.State) int {
	switch state {
	case process.Starting:
		return faults.StillRunning
	case process.Running:
		return faults.Success
	case process.Stopping:
		return faults.StillRunning
	case process.Stopped:
		return faults.NotRunning
	case process.Backoff:
		return faults.StillRunning
	case process.Exited:
		return faults.NotRunning
	case process.Fatal:
		return faults.Failed
	case process.Unknown:
		return faults.Failed
	}
	return faults.Failed
}

func getProcessInfo(nodename string, proc *process.Process) *types.ProcessInfo {

	return &types.ProcessInfo{
		Node:          nodename,
		Name:          proc.GetName(),
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

func (s *Supervisor) getNodeName() string {

	nodename, _ := os.Hostname()

	httpServerConfig, ok := s.activeConfig.GetInetHTTPServer()
	if ok {
		nodename = httpServerConfig.GetString("nodename", nodename)
	}
	return nodename
}

// GetAllProcessInfo get all the program information managed by supervisor
func (s *Supervisor) GetAllProcessInfo(r *http.Request, args *struct{}, reply *struct{ AllProcessInfo []types.ProcessInfo }) error {
	reply.AllProcessInfo = make([]types.ProcessInfo, 0)
	s.procMgr.ForEachProcess(func(proc *process.Process) {
		procInfo := getProcessInfo(s.getNodeName(), proc)
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
		return faults.NewFault(faults.BadName, fmt.Sprintf("no process named %s", args.Name))
	}

	reply.ProcInfo = *getProcessInfo(s.getNodeName(), proc)
	return nil
}

// StartProcess start the given program
func (s *Supervisor) StartProcess(r *http.Request, args *StartProcessArgs, reply *struct{ Success bool }) error {
	procs := s.procMgr.FindMatch(args.Name)

	if len(procs) <= 0 {
		return faults.NewFault(faults.BadName, fmt.Sprintf("no process named %s", args.Name))
	}

	for _, proc := range procs {
		if proc.IsRunning() && len(procs) == 1 {
			reply.Success = false
			return faults.NewFault(faults.AlreadyStarted, fmt.Sprintf("process %s is already running", args.Name))
		}
		proc.Start(args.Wait)
	}

	for _, proc := range procs {
		if !proc.IsRunning() {
			log.WithFields(log.Fields{"program": proc.GetName()}).Error("fail to start process")
			if len(procs) == 1 {
				return faults.NewFault(faults.Failed, fmt.Sprintf("Fail to start process %s", proc.GetName()))
			}
		}
	}

	reply.Success = true
	return nil
}

// StartAllProcesses start all the programs
func (s *Supervisor) StartAllProcesses(r *http.Request, args *struct {
	Wait bool `default:"true"`
}, reply *struct{ ProcessStatuses []types.ProcessStatus }) error {

	finishedProcCh := make(chan *process.Process)

	n := s.procMgr.AsyncForEachProcess(func(proc *process.Process) {
		if !proc.IsRunning() {
			proc.Start(args.Wait)
		}
	}, finishedProcCh)

	for range n {
		proc, _ := <-finishedProcCh
		reply.ProcessStatuses = append(reply.ProcessStatuses, types.ProcessStatus{
			Name:        proc.GetName(),
			Group:       proc.GetGroup(),
			Status:      faults.Failed,
			Description: "Fail to start",
		})
	}
	return nil
}

// StartProcessGroup start all the processes in one group
func (s *Supervisor) StartProcessGroup(r *http.Request, args *StartProcessArgs, reply *struct{ ProcessStatuses []types.ProcessStatus }) error {
	log.WithFields(log.Fields{"group": args.Name}).Info("start process group")
	finishedProcCh := make(chan *process.Process)

	n := s.procMgr.AsyncForEachProcess(func(proc *process.Process) {
		if proc.GetGroup() == args.Name {
			proc.Start(args.Wait)
		}
	}, finishedProcCh)

	for range n {
		proc, _ := <-finishedProcCh
		if proc.GetGroup() == args.Name && !proc.IsRunning() {
			reply.ProcessStatuses = append(reply.ProcessStatuses, types.ProcessStatus{
				Name:        proc.GetName(),
				Group:       proc.GetGroup(),
				Status:      faults.Failed,
				Description: "Fail to start",
			})
		}
	}

	return nil
}

// StopProcess stop given program
func (s *Supervisor) StopProcess(r *http.Request, args *StartProcessArgs, reply *struct{ Success bool }) error {
	log.WithFields(log.Fields{"program": args.Name}).Info("stop process")
	procs := s.procMgr.FindMatch(args.Name)
	if len(procs) <= 0 {
		return faults.NewFault(faults.BadName, fmt.Sprintf("no process named %s", args.Name))
	}
	for _, proc := range procs {
		if !proc.IsRunning() && len(procs) == 1 {
			return faults.NewFault(faults.NotRunning, fmt.Sprintf("Process %s is stopped already", args.Name))
		}
		proc.Stop(args.Wait)
	}
	for _, proc := range procs {
		if proc.IsRunning() {
			log.WithFields(log.Fields{"program": proc.GetName()}).Error("fail to stop process")
			if len(procs) == 1 {
				return faults.NewFault(faults.Failed, fmt.Sprintf("Fail to stop process %s", proc.GetName()))
			}
		}
	}
	reply.Success = true
	return nil
}

// StopProcessGroup stop all processes in one group
func (s *Supervisor) StopProcessGroup(r *http.Request, args *StartProcessArgs, reply *struct{ ProcessStatuses []types.ProcessStatus }) error {
	log.WithFields(log.Fields{"group": args.Name}).Info("stop process group")
	finishedProcCh := make(chan *process.Process)

	n := s.procMgr.AsyncForEachProcess(func(proc *process.Process) {
		if proc.GetGroup() == args.Name && !proc.IsRunning() {

			proc.Stop(args.Wait)
		}
	}, finishedProcCh)

	for range n {
		proc, ok := <-finishedProcCh
		if ok && proc.GetGroup() == args.Name && proc.IsRunning() {
			reply.ProcessStatuses = append(reply.ProcessStatuses, types.ProcessStatus{
				Name:        proc.GetName(),
				Group:       proc.GetGroup(),
				Status:      faults.Failed,
				Description: "Fail to stop",
			})
		}
	}
	return nil
}

// StopAllProcesses stop all programs managed by supervisor
func (s *Supervisor) StopAllProcesses(r *http.Request, args *struct {
	Wait bool `default:"true"`
}, reply *struct{ ProcessStatuses []types.ProcessStatus }) error {
	finishedProcCh := make(chan *process.Process)

	n := s.procMgr.AsyncForEachProcess(func(proc *process.Process) {
		if proc.IsRunning() {
			proc.Stop(args.Wait)

		}
	}, finishedProcCh)

	for range n {
		proc, _ := <-finishedProcCh
		if proc.IsRunning() {
			reply.ProcessStatuses = append(reply.ProcessStatuses, types.ProcessStatus{
				Name:        proc.GetName(),
				Group:       proc.GetGroup(),
				Status:      faults.Failed,
				Description: "Fail to start",
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
		return fmt.Errorf("no process named %s", args.Name)
	}
	for _, proc := range procs {
		_ = proc.Signal(args.Signal, true)
	}
	reply.Success = true
	return nil
}

// SignalProcessGroup send signal to all processes in one group
func (s *Supervisor) SignalProcessGroup(r *http.Request, args *types.ProcessSignal, reply *struct{ ProcessStatuses []types.ProcessStatus }) error {
	s.procMgr.ForEachProcess(func(proc *process.Process) {
		if proc.GetGroup() == args.Name {
			err := proc.Signal(args.Signal, true)

			status := faults.Success
			description := "OK"
			if err != nil {
				status = faults.Failed
				description = "Failed"
			}
			reply.ProcessStatuses = append(reply.ProcessStatuses, types.ProcessStatus{
				Name:        proc.GetName(),
				Group:       proc.GetGroup(),
				Status:      status,
				Description: description,
			})
		}
	})

	return nil
}

// SignalAllProcesses send signal to all the processes in the supervisor
func (s *Supervisor) SignalAllProcesses(r *http.Request, args *struct{ Signal string }, reply *struct{ ProcessStatuses []types.ProcessStatus }) error {
	s.procMgr.ForEachProcess(func(proc *process.Process) {
		err := proc.Signal(args.Signal, true)
		status := faults.Success
		description := "OK"
		if err != nil {
			status = faults.Failed
			description = "Failed"
		}
		reply.ProcessStatuses = append(reply.ProcessStatuses, types.ProcessStatus{
			Name:        proc.GetName(),
			Group:       proc.GetGroup(),
			Status:      status,
			Description: description,
		})
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
func (s *Supervisor) Reload(action ReloadAction) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	newConfig := config.NewConfig(s.activeConfig.GetConfigFile())

	loadedPrograms, err := newConfig.Load()

	if err != nil {
		log.Error("failed to load config: ", err)
		return err
	}

	log.WithFields(log.Fields{"programs": strings.Join(loadedPrograms, ",")}).Info("loaded programs")

	// get the previous loaded programs

	prevConfig := s.activeConfig
	s.activeConfig = newConfig

	if checkErr := s.checkRequiredResources(); checkErr != nil {
		log.Error(checkErr)
		os.Exit(1)
	}

	s.setSupervisordInfo()

	prevEntries := prevConfig.GetAllEntries()

	// Remove the programs that are removed from configuration file
	for _, prevEntry := range prevEntries {
		if prevEntry.IsProgram() {
			program := prevEntry.GetProgramName()
			if newEntry, ok := newConfig.GetEntry(program); !ok || !newEntry.IsProgram() || !prevEntry.IsSame(newEntry) {
				log.WithFields(log.Fields{"program": program}).Info("the program is removed and will be stopped")
				s.activeConfig.RemoveProgram(program)
				proc := s.procMgr.Remove(program)
				if proc != nil {
					proc.Stop(false)
				}
			}
		}
	}

	currentEntries := newConfig.GetAllEntries()

	// Add the new programs that are added in configuration file
	for _, currentEntry := range currentEntries {
		if currentEntry.IsProgram() {
			program := currentEntry.GetProgramName()
			proc := s.procMgr.Find(program)
			// If the program is not exist, create it. If the program is exist and the action is RestartAll, restart it.
			if proc == nil {
				log.WithFields(log.Fields{"program": program}).Info("the program is added and will be created")
				proc = s.procMgr.CreateProcess(s.GetSupervisorID(), currentEntry)
			} else if action == RestartAll && proc.IsRunning() {
				log.WithFields(log.Fields{"program": program}).Info("the program will be restarted")
				proc.Stop(true)
				proc.Start(true)
			}
		}
	}

	s.startEventListeners()
	s.startHTTPServer()
	s.startAutoStartPrograms()

	return nil

}

// WaitForExit waits for supervisord to exit
func (s *Supervisor) WaitForExit() {
	for {
		/*f s.IsRestarting() {
			s.procMgr.StopAllProcesses()
			break
		}*/

		if s.procMgr.IsAllProcessesStopped() {
			break
		}

		time.Sleep(2 * time.Second)
	}
}

func (s *Supervisor) startAutoStartPrograms() {
	s.procMgr.StartAutoStartPrograms()
}

func (s *Supervisor) startEventListeners() {
	eventListeners := s.activeConfig.GetEventListeners()
	for _, entry := range eventListeners {
		proc := s.procMgr.CreateProcess(s.GetSupervisorID(), entry)
		proc.Start(false)
	}

	if len(eventListeners) > 0 {
		time.Sleep(1 * time.Second)
	}
}

func (s *Supervisor) startHTTPServer() {
	httpServerConfig, ok := s.activeConfig.GetInetHTTPServer()
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
				s.getRemoteNodes(),
				func() {
					cond.L.Lock()
					cond.Signal()
					cond.L.Unlock()
				})
			cond.Wait()
		}
	}

	httpServerConfig, ok = s.activeConfig.GetUnixHTTPServer()
	if ok {
		env := config.NewStringExpression("here", s.activeConfig.GetConfigFileDir())
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

func (s *Supervisor) getRemoteNodes() map[string]*NodeLoginInfo {

	result := make(map[string]*NodeLoginInfo)

	httpServerConfig, exist := s.activeConfig.GetInetHTTPServer()

	if !exist {
		log.Warn("inet_http_server section not found in config file")
		return result
	}

	for i := 1; i < 1000; i++ {
		port := httpServerConfig.GetString("remote_"+strconv.Itoa(i)+"_port", "")
		if port == "" {
			continue
		}

		node := httpServerConfig.GetString("remote_"+strconv.Itoa(i)+"_node", "")
		if node == "" {
			node = port
		}

		user := httpServerConfig.GetString("remote_"+strconv.Itoa(i)+"_user", "")
		password := httpServerConfig.GetString("remote_"+strconv.Itoa(i)+"_password", "")
		if port != "" {
			result[node] = NewNodeLoginInfo(node, "http://"+port, user, password)
		}
		log.WithFields(log.Fields{"node": node, "port": port, "user": user}).Info("remote node configured")
	}

	return result
}

func (s *Supervisor) GetPidFile() string {
	supervisordConf, ok := s.activeConfig.GetSupervisord()
	if ok {
		env := config.NewStringExpression("here", s.activeConfig.GetConfigFileDir())
		pidfile, err := env.Eval(supervisordConf.GetString("pidfile", "supervisord.pid"))
		if err == nil {
			return pidfile
		}
	}
	return "supervisord.pid"
}
func (s *Supervisor) setSupervisordInfo() {
	supervisordConf, ok := s.activeConfig.GetSupervisord()
	if ok {
		// set supervisord log

		env := config.NewStringExpression("here", s.activeConfig.GetConfigFileDir())
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
			fileNameWithTimestamp := supervisordConf.GetBool("logfile_timestamp_suffix", true)
			loglevel := supervisordConf.GetString("loglevel", "info")
			props := make(map[string]string)
			s.logger = logger.NewLogger("supervisord", logFile, &sync.Mutex{}, logfileMaxbytes, logfileBackups, fileNameWithTimestamp, props, logEventEmitter)
			log.SetLevel(toLogLevel(loglevel))
			log.SetFormatter(&log.TextFormatter{DisableColors: true, FullTimestamp: true})
			log.SetOutput(logger.NewReformatLog(s.logger))
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
func (s *Supervisor) ReloadConfig(r *http.Request, args *struct{}, reply *struct{ Value [][][]string }) error {
	log.Info("start to reload config")
	s.pendingConfig = config.NewConfig(s.activeConfig.GetConfigFile())

	_, err := s.pendingConfig.Load()
	if err != nil {
		log.Error("failed to load config: ", err)
		return err
	}

	s.lock.Lock()
	defer s.lock.Unlock()

	result, err := s.pendingConfig.Sub(s.activeConfig)

	if err != nil {
		log.Error("failed to compare config: ", err)
		return err
	}

	reply.Value = [][][]string{[][]string{result.AddedGroups, result.ChangedGroups, result.RemovedGroups}}

	return err
}

// AddProcessGroup adds a process group to the supervisor
func (s *Supervisor) AddProcessGroup(r *http.Request, args *struct{ Name string }, reply *struct{ Success bool }) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	reply.Success = false

	programEntries := s.pendingConfig.GetProgramEntriesInGroup(args.Name)

	for _, programEntry := range programEntries {
		reply.Success = true
		oldEntry, oldExist := s.activeConfig.GetEntry(programEntry.GetProgramName())
		if !oldExist || !programEntry.IsSame(oldEntry) {
			log.WithFields(log.Fields{"program": programEntry.GetProgramName(), "group": args.Name}).Info("the program is added or changed and will be created")
			proc := s.procMgr.Remove(programEntry.GetProgramName())
			// stop the old one
			if proc != nil {
				proc.Stop(true)
			}
			s.activeConfig.AddEntry(programEntry)

			// create the new one but not start it, the client should call StartProcessGroup to start it
			_ = s.procMgr.CreateProcess(s.GetSupervisorID(), programEntry)

		}
	}
	if len(programEntries) > 0 {
		return nil
	}
	return fmt.Errorf("no such group %s", args.Name)
}

// RemoveProcessGroup removes a process group from the supervisor
func (s *Supervisor) RemoveProcessGroup(r *http.Request, args *struct{ Name string }, reply *struct{ Success bool }) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	reply.Success = false

	gprogramEntries := s.activeConfig.GetProgramEntriesInGroup(args.Name)

	for _, programEntry := range gprogramEntries {
		reply.Success = true
		log.WithFields(log.Fields{"program": programEntry.GetProgramName(), "group": args.Name}).Info("the program is removed and will be stopped")
		s.activeConfig.RemoveProgram(programEntry.GetProgramName())
		proc := s.procMgr.Remove(programEntry.GetProgramName())
		if proc != nil {
			proc.Stop(true)
		}
	}
	if len(gprogramEntries) > 0 {
		return nil
	}

	return fmt.Errorf("no such group %s", args.Name)
}

// ReadProcessStdoutLog reads stdout of given program
func (s *Supervisor) ReadProcessStdoutLog(r *http.Request, args *ProcessLogReadInfo, reply *struct{ LogData string }) error {
	proc := s.procMgr.Find(args.Name)
	if proc == nil {
		return fmt.Errorf("no such process %s", args.Name)
	}
	var err error
	reply.LogData, err = proc.StdoutLog.ReadLog(int64(args.Offset), int64(args.Length))
	return err
}

// ReadProcessStderrLog reads stderr log of given program
func (s *Supervisor) ReadProcessStderrLog(r *http.Request, args *ProcessLogReadInfo, reply *struct{ LogData string }) error {
	proc := s.procMgr.Find(args.Name)
	if proc == nil {
		return fmt.Errorf("no such process %s", args.Name)
	}
	var err error
	reply.LogData, err = proc.StderrLog.ReadLog(int64(args.Offset), int64(args.Length))
	return err
}

// TailProcessStdoutLog tails stdout of the program
func (s *Supervisor) TailProcessStdoutLog(r *http.Request, args *ProcessLogReadInfo, reply *ProcessTailLog) error {
	proc := s.procMgr.Find(args.Name)
	if proc == nil {
		return fmt.Errorf("no such process %s", args.Name)
	}
	var err error
	var offset int64
	reply.LogData, offset, reply.Overflow, err = proc.StdoutLog.ReadTailLog(int64(args.Offset), int64(args.Length))
	reply.Offset = int(offset)
	return err
}

// TailProcessStderrLog tails stderr of the program
func (s *Supervisor) TailProcessStderrLog(r *http.Request, args *ProcessLogReadInfo, reply *ProcessTailLog) error {
	proc := s.procMgr.Find(args.Name)
	if proc == nil {
		return fmt.Errorf("no such process %s", args.Name)
	}
	var err error
	var offset int64
	reply.LogData, offset, reply.Overflow, err = proc.StderrLog.ReadTailLog(int64(args.Offset), int64(args.Length))
	reply.Offset = int(offset)
	return err
}

// ClearProcessLogs clears log of given program
func (s *Supervisor) ClearProcessLogs(r *http.Request, args *struct{ Name string }, reply *struct{ Success bool }) error {

	procs := s.procMgr.FindMatch(args.Name)

	if len(procs) <= 0 {
		reply.Success = false
		return fmt.Errorf("no such process %s", args.Name)
	}

	for _, proc := range procs {
		_ = proc.StdoutLog.ClearAllLogFile()
		_ = proc.StderrLog.ClearAllLogFile()
	}
	reply.Success = true
	return nil

}

// ClearAllProcessLogs clears logs of all programs
func (s *Supervisor) ClearAllProcessLogs(r *http.Request, args *struct{}, reply *struct{ ProcessStatuses []types.ProcessStatus }) error {

	s.procMgr.ForEachProcess(func(proc *process.Process) {
		_ = proc.StdoutLog.ClearAllLogFile()
		_ = proc.StderrLog.ClearAllLogFile()

		reply.ProcessStatuses = append(reply.ProcessStatuses, types.ProcessStatus{
			Name:        proc.GetName(),
			Group:       proc.GetGroup(),
			Status:      faults.Success,
			Description: "Suceed to clear",
		})
	})

	return nil
}

// GetManager get the Manager object created by supervisor
func (s *Supervisor) GetManager() *process.Manager {
	return s.procMgr
}
