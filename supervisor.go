package gopm

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/hashicorp/go-memdb"
	"github.com/stuartcarnie/gopm/config"
	"github.com/stuartcarnie/gopm/faults"
	"github.com/stuartcarnie/gopm/logger"
	"github.com/stuartcarnie/gopm/process"
	"github.com/stuartcarnie/gopm/rpc"
	"github.com/stuartcarnie/gopm/types"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

const (
	// SupervisorVersion the version of supervisor
	SupervisorVersion = "3.0"
)

// Supervisor manage all the processes defined in the supervisor configuration file.
// All the supervisor public interface is defined in this class
type Supervisor struct {
	configFile string
	config     *config.Config   // supervisor configuration
	procMgr    *process.Manager // process manager
	httpServer *http.Server
	grpc       *grpc.Server
	logger     logger.Logger // logger manager
	restarting bool          // if supervisor is in restarting state
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
	return &Supervisor{
		configFile: configFile,
		config:     config.NewConfig(),
		procMgr:    process.NewManager(),
		restarting: false,
	}
}

// GetSupervisorID get the supervisor identifier from configuration file
func (s *Supervisor) GetSupervisorID() string {
	return "supervisor"
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

// Restart restart the supervisor
func (s *Supervisor) Restart(r *http.Request, args *struct{}, reply *struct{ Ret bool }) error {
	zap.L().Info("Restart requested")

	s.restarting = true
	reply.Ret = true
	return nil
}

// IsRestarting check if supervisor is in restarting state
func (s *Supervisor) IsRestarting() bool {
	return s.restarting
}

func getProcessInfo(proc *process.Process) *types.ProcessInfo {
	return &types.ProcessInfo{
		Name:          proc.Name(),
		Group:         proc.Group(),
		Description:   proc.Description(),
		Start:         proc.StartTime().Unix(),
		Stop:          proc.StopTime().Unix(),
		Now:           time.Now().Unix(),
		State:         int64(proc.State()),
		StateName:     proc.State().String(),
		SpawnErr:      "",
		ExitStatus:    int64(proc.ExitStatus()),
		Logfile:       proc.StdoutLogfile(),
		StdoutLogfile: proc.StdoutLogfile(),
		StderrLogfile: proc.StderrLogfile(),
		Pid:           int64(proc.Pid()),
	}
}

// GetAllProcessInfo get all the program informations managed by supervisor
func (s *Supervisor) GetAllProcessInfo(r *http.Request, args *struct{}, reply *struct{ AllProcessInfo []types.ProcessInfo }) error {
	var pi types.ProcessInfos
	s.procMgr.ForEachProcess(func(proc *process.Process) {
		procInfo := getProcessInfo(proc)
		pi = append(pi, *procInfo)
	})

	pi.SortByName()
	reply.AllProcessInfo = pi

	return nil
}

// StartProcessGroup start all the processes in one group
func (s *Supervisor) StartProcessGroup(r *http.Request, args *StartProcessArgs, reply *struct{ AllProcessInfo []types.ProcessInfo }) error {
	zap.L().Info("start process group", zap.String("group", args.Name))
	finishedProcCh := make(chan *process.Process)

	n := s.procMgr.AsyncForEachProcess(func(proc *process.Process) {
		if proc.Group() == args.Name {
			proc.Start(args.Wait)
		}
	}, finishedProcCh)

	for i := 0; i < n; i++ {
		proc, ok := <-finishedProcCh
		if ok && proc.Group() == args.Name {
			reply.AllProcessInfo = append(reply.AllProcessInfo, *getProcessInfo(proc))
		}
	}

	return nil
}

// StopProcessGroup stop all processes in one group
func (s *Supervisor) StopProcessGroup(r *http.Request, args *StartProcessArgs, reply *struct{ AllProcessInfo []types.ProcessInfo }) error {
	zap.L().Info("stop process group", zap.String("group", args.Name))
	finishedProcCh := make(chan *process.Process)
	n := s.procMgr.AsyncForEachProcess(func(proc *process.Process) {
		if proc.Group() == args.Name {
			proc.Stop(args.Wait)
		}
	}, finishedProcCh)

	for i := 0; i < n; i++ {
		proc, ok := <-finishedProcCh
		if ok && proc.Group() == args.Name {
			reply.AllProcessInfo = append(reply.AllProcessInfo, *getProcessInfo(proc))
		}
	}
	return nil
}

// SendProcessStdin send data to program through stdin
func (s *Supervisor) SendProcessStdin(r *http.Request, args *ProcessStdin, reply *struct{ Success bool }) error {
	proc := s.procMgr.Find(args.Name)
	if proc == nil {
		zap.L().Error("program does not exist", zap.String("program", args.Name))
		return fmt.Errorf("NOT_RUNNING")
	}
	if proc.State() != process.Running {
		zap.L().Error("program does not run", zap.String("program", args.Name))
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

// Reload reload the supervisor configuration
//return err, addedGroup, changedGroup, removedGroup
//
func (s *Supervisor) Reload() (addedGroup, changedGroup, removedGroup []string, err error) {
	changes, err := s.config.LoadPath(s.configFile)
	if len(changes) == 0 {
		return nil, nil, nil, nil
	}

	if err != nil {
		var el Errors
		if errors.As(err, &el) {
			errs := el.Errors()
			zap.L().Error("Error loading configuration")
			for _, err := range errs {
				zap.L().Error("Configuration file error", zap.Error(err))
			}
		} else {
			zap.L().Error("Error loading configuration", zap.Error(err))
		}

		return nil, nil, nil, err
	}

	s.createPrograms(changes)
	s.startHTTPServer(changes)
	s.startGrpcServer(changes)
	s.startAutoStartPrograms()

	return nil, nil, nil, err
}

// WaitForExit wait the supervisor to exit
func (s *Supervisor) WaitForExit() {
	for {
		if s.IsRestarting() {
			s.procMgr.StopAllProcesses()
			break
		}
		time.Sleep(10 * time.Second)
	}
}

func (s *Supervisor) createPrograms(changes memdb.Changes) {
	for _, ch := range changes {
		if ch.Table != "process" {
			continue
		}

		switch {
		case ch.Created(), ch.Updated():
			s.procMgr.CreateOrUpdateProcess(s.GetSupervisorID(), ch.After.(*config.Process))

		case ch.Deleted():
			proc := s.procMgr.Remove(ch.Before.(*config.Process).Name)
			if proc != nil {
				proc.Destroy()
			}
		}
	}
}

func (s *Supervisor) startAutoStartPrograms() {
	s.procMgr.StartAutoStartPrograms()
}

func (s *Supervisor) findServerChange(name string, changes memdb.Changes) *memdb.Change {
	for i := range changes {
		ch := &changes[i]
		if ch.Table != "server" {
			continue
		}

		var id string
		if ch.Deleted() {
			id = ch.Before.(*config.Server).Name
		} else {
			id = ch.After.(*config.Server).Name
		}
		if id == name {
			return ch
		}
	}
	return nil
}

func (s *Supervisor) startHTTPServer(changes memdb.Changes) {
	found := s.findServerChange("http", changes)
	if found == nil {
		return
	}

	var cfg *config.Server
	if found.Updated() || found.Created() {
		cfg = found.After.(*config.Server)
	}

	go func() {
		if s.httpServer != nil {
			err := s.httpServer.Shutdown(context.Background())
			if err != nil {
				zap.L().Error("Unable to shutdown HTTP server", zap.Error(err))
			} else {
				zap.L().Info("Stopped HTTP server")
			}
			s.httpServer = nil
		}

		if cfg == nil {
			return
		}

		mux := http.NewServeMux()
		svc := NewSupervisorRestful(s)
		progRestHandler := svc.CreateProgramHandler()
		mux.Handle("/program/", progRestHandler)
		supervisorRestHandler := svc.CreateSupervisorHandler()
		mux.Handle("/supervisor/", supervisorRestHandler)
		webguiHandler := NewSupervisorWebgui(s).CreateHandler()
		mux.Handle("/", webguiHandler)

		zap.L().Info("Starting HTTP server", zap.String("addr", cfg.Address))
		srv := http.Server{Handler: mux, Addr: cfg.Address}
		s.httpServer = &srv
		err := srv.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			zap.L().Error("Unable to start HTTP server", zap.Error(err))
		}
	}()
}

func (s *Supervisor) startGrpcServer(changes memdb.Changes) {
	// restart asynchronously to permit existing Reload request to complete
	found := s.findServerChange("grpc", changes)
	if found == nil {
		return
	}

	var cfg *config.Server
	if found.Updated() || found.Created() {
		cfg = found.After.(*config.Server)
	}

	go func() {
		if s.grpc != nil {
			s.grpc.GracefulStop()
			zap.L().Info("Stopped gRPC server")
			s.grpc = nil
		}

		if cfg == nil {
			return
		}

		ln, err := net.Listen("tcp", cfg.Address)
		if err != nil {
			zap.L().Error("Unable to start gRPC", zap.Error(err), zap.String("addr", cfg.Address))
			return
		}

		grpcServer := grpc.NewServer()
		rpc.RegisterGopmServer(grpcServer, s)
		reflection.Register(grpcServer)
		s.grpc = grpcServer

		zap.L().Info("Starting gRPC server", zap.String("addr", cfg.Address))
		err = grpcServer.Serve(ln)
		if err != nil && err != io.EOF {
			zap.L().Error("Unable to start gRPC server", zap.Error(err))
		}
	}()
}

// ReadProcessStdoutLog read the stdout log of a given program
func (s *Supervisor) ReadProcessStdoutLog(r *http.Request, args *ProcessLogReadInfo, reply *struct{ LogData string }) error {
	proc := s.procMgr.Find(args.Name)
	if proc == nil {
		return fmt.Errorf("no such process: %s", args.Name)
	}
	var err error
	reply.LogData, err = proc.StdoutLog.ReadLog(int64(args.Offset), int64(args.Length))
	return err
}

// ReadProcessStderrLog read the stderr log of a given program
func (s *Supervisor) ReadProcessStderrLog(r *http.Request, args *ProcessLogReadInfo, reply *struct{ LogData string }) error {
	proc := s.procMgr.Find(args.Name)
	if proc == nil {
		return fmt.Errorf("no such process: %s", args.Name)
	}
	var err error
	reply.LogData, err = proc.StderrLog.ReadLog(int64(args.Offset), int64(args.Length))
	return err
}

// TailProcessStdoutLog tail the stdout of a program
func (s *Supervisor) TailProcessStdoutLog(r *http.Request, args *ProcessLogReadInfo, reply *ProcessTailLog) error {
	proc := s.procMgr.Find(args.Name)
	if proc == nil {
		return fmt.Errorf("no such process: %s", args.Name)
	}
	var err error
	reply.LogData, reply.Offset, reply.Overflow, err = proc.StdoutLog.ReadTailLog(int64(args.Offset), int64(args.Length))
	return err
}

// TailProcessStderrLog tail the stderr of a program
func (s *Supervisor) TailProcessStderrLog(r *http.Request, args *ProcessLogReadInfo, reply *ProcessTailLog) error {
	proc := s.procMgr.Find(args.Name)
	if proc == nil {
		return fmt.Errorf("no such process: %s", args.Name)
	}
	var err error
	reply.LogData, reply.Offset, reply.Overflow, err = proc.StderrLog.ReadTailLog(int64(args.Offset), int64(args.Length))
	return err
}

// ClearProcessLogs clear the log of a given program
func (s *Supervisor) ClearProcessLogs(r *http.Request, args *struct{ Name string }, reply *struct{ Success bool }) error {
	proc := s.procMgr.Find(args.Name)
	if proc == nil {
		return fmt.Errorf("no such process: %s", args.Name)
	}
	err1 := proc.StdoutLog.ClearAllLogFile()
	err2 := proc.StderrLog.ClearAllLogFile()
	reply.Success = err1 == nil && err2 == nil
	if err1 != nil {
		return err1
	}
	return err2
}

// ClearAllProcessLogs clear the logs of all programs
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

// GetManager get the Manager object created by superisor
func (s *Supervisor) GetManager() *process.Manager {
	return s.procMgr
}
