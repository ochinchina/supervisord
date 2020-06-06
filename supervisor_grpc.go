package gopm

import (
	"context"
	"os"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/stuartcarnie/gopm/logger"
	"github.com/stuartcarnie/gopm/process"
	"github.com/stuartcarnie/gopm/rpc"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var _ rpc.GopmServer = (*Supervisor)(nil)

func (s *Supervisor) GetVersion(_ context.Context, _ *empty.Empty) (*rpc.VersionResponse, error) {
	return &rpc.VersionResponse{Version: SupervisorVersion}, nil
}

func getRpcProcessInfo(proc *process.Process) *rpc.ProcessInfo {
	return &rpc.ProcessInfo{
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

func (s *Supervisor) GetProcessInfo(_ context.Context, _ *empty.Empty) (*rpc.ProcessInfoResponse, error) {
	var res rpc.ProcessInfoResponse
	s.procMgr.ForEachProcess(func(proc *process.Process) {
		res.Processes = append(res.Processes, getRpcProcessInfo(proc))
	})
	return &res, nil
}

func (s *Supervisor) StartProcess(_ context.Context, req *rpc.StartStopRequest) (*rpc.StartStopResponse, error) {
	procs := s.procMgr.FindMatch(req.Name)

	if len(procs) <= 0 {
		return nil, status.Error(codes.NotFound, "Process not found")
	}

	for _, proc := range procs {
		proc.Start(req.Wait)
	}

	return &rpc.StartStopResponse{}, nil
}

func (s *Supervisor) StopProcess(_ context.Context, req *rpc.StartStopRequest) (*rpc.StartStopResponse, error) {
	procs := s.procMgr.FindMatch(req.Name)
	if len(procs) <= 0 {
		return nil, status.Error(codes.NotFound, "Process not found")
	}
	for _, proc := range procs {
		proc.Stop(req.Wait)
	}

	return &rpc.StartStopResponse{}, nil
}

func (s *Supervisor) StartAllProcesses(_ context.Context, req *rpc.StartStopAllRequest) (*rpc.ProcessInfoResponse, error) {
	var (
		g     errgroup.Group
		procs []*process.Process
	)

	s.procMgr.ForEachProcess(func(p *process.Process) {
		procs = append(procs, p)
		g.Go(func() error {
			p.Start(req.Wait)
			return nil
		})
	})
	_ = g.Wait()

	var res rpc.ProcessInfoResponse
	for _, proc := range procs {
		res.Processes = append(res.Processes, getRpcProcessInfo(proc))
	}

	return &res, nil
}

func (s *Supervisor) StopAllProcesses(_ context.Context, req *rpc.StartStopAllRequest) (*rpc.ProcessInfoResponse, error) {
	var (
		g     errgroup.Group
		procs []*process.Process
	)

	s.procMgr.ForEachProcess(func(p *process.Process) {
		procs = append(procs, p)
		g.Go(func() error {
			p.Stop(req.Wait)
			return nil
		})
	})
	_ = g.Wait()

	var res rpc.ProcessInfoResponse
	for _, proc := range procs {
		res.Processes = append(res.Processes, getRpcProcessInfo(proc))
	}

	return &res, nil
}

func (s *Supervisor) Shutdown(context.Context, *empty.Empty) (*empty.Empty, error) {
	// TODO(sgc): This is not right
	s.procMgr.StopAllProcesses()
	go func() {
		time.Sleep(1 * time.Second)
		os.Exit(0)
	}()

	return &empty.Empty{}, nil
}

func (s *Supervisor) ReloadConfig(context.Context, *empty.Empty) (*rpc.ReloadConfigResponse, error) {
	addedGroup, changedGroup, removedGroup, err := s.Reload()
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &rpc.ReloadConfigResponse{
		AddedGroup:   addedGroup,
		ChangedGroup: changedGroup,
		RemovedGroup: removedGroup,
	}, nil
}

func (s *Supervisor) TailLog(req *rpc.TailLogRequest, stream rpc.Gopm_TailLogServer) error {
	procMgr := s.GetManager()
	proc := procMgr.Find(req.Name)
	if proc == nil {
		return status.Error(codes.NotFound, "Process not found")
	}

	var ok bool
	var compositeLogger *logger.CompositeLogger = nil
	if req.Device == rpc.LogDevice_Stdout {
		compositeLogger, ok = proc.StdoutLog.(*logger.CompositeLogger)
	} else {
		compositeLogger, ok = proc.StderrLog.(*logger.CompositeLogger)
	}
	if ok {
		ch := make(chan []byte, 100)
		clog := logger.NewChanLogger(ch)
		compositeLogger.AddLogger(clog)
		var (
			res rpc.TailLogResponse
			ctx = stream.Context()
		)

	READ:
		for {
			select {
			case buf := <-ch:
				res.Lines = buf
				err := stream.Send(&res)
				clog.PutBuffer(buf)
				if err != nil {
					break READ
				}

			case <-ctx.Done():
				break READ
			}
		}
		compositeLogger.RemoveLogger(clog)
		_ = clog.Close()
	}
	return nil
}

func (s *Supervisor) SignalProcess(_ context.Context, req *rpc.SignalProcessRequest) (*empty.Empty, error) {
	procs := s.procMgr.FindMatch(req.Name)
	if len(procs) <= 0 {
		return nil, status.Error(codes.NotFound, "Process not found")
	}
	sig, err := req.Signal.ToSignal()
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "Invalid signal")
	}

	for _, proc := range procs {
		// TODO(sgc): collect errors?
		_ = proc.Signal(sig, false)
	}

	return &empty.Empty{}, nil
}

func (s *Supervisor) SignalProcessGroup(_ context.Context, req *rpc.SignalProcessRequest) (*rpc.ProcessInfoResponse, error) {
	sig, err := req.Signal.ToSignal()
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "Invalid signal")
	}

	var res rpc.ProcessInfoResponse
	s.procMgr.ForEachProcess(func(proc *process.Process) {
		if proc.Group() == req.Name {
			proc.Signal(sig, false)
			res.Processes = append(res.Processes, getRpcProcessInfo(proc))
		}
	})

	return &res, nil
}

func (s *Supervisor) SignalAllProcesses(_ context.Context, req *rpc.SignalProcessRequest) (*rpc.ProcessInfoResponse, error) {
	sig, err := req.Signal.ToSignal()
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "Invalid signal")
	}

	var res rpc.ProcessInfoResponse
	s.procMgr.ForEachProcess(func(proc *process.Process) {
		proc.Signal(sig, false)
		res.Processes = append(res.Processes, getRpcProcessInfo(proc))
	})

	return &res, nil
}
