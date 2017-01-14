package main

import (
	"fmt"
	"net/http"
	"os"
	"time"
)

const (
	VERSION = "3.0"
)

type Supervisor struct {
	config *Config
	procMgr *ProcessManager
	xmlRPC *XmlRPC
}

type processInfo struct {
	Name string `xml:"name"`
	Group string `xml:"group"`
	Description string `xml:"description"`
	Start int64 `xml:"start"`
	Stop int64 `xml:"stop"`
	Now int64 `xml:"now"`
	State int `xml:"state"`
	Statename string `xml:"statename"`
	Spawnerr string `xml:"spawnerr"`
	Exitstatus int `xml:"exitstatus"`
	Logfile string `xml:"logfile"`
	Stdout_logfile string `xml:"stdout_logfile"`
	Stderr_logfile string `xml:"stderr_logfile"`
	Pid int `xml:"pid"`
}

type StartProcessArgs struct {
	Name string
	Wait bool `default:"true"`
}

type processSignal struct {
	Name string
	Signal string
}

type processStdin struct {
	Name string
	Chars string
}

type remoteCommEvent struct {
	Type string
	Data string
}

type processStateInfo struct {
	Statecode int
	Statename string
}

type logReadInfo struct {
	Offset int
	Length int
}

func NewSupervisor( configFile string ) *Supervisor {
	return &Supervisor{ config: NewConfig( configFile ),
			procMgr: newProcessManager(),
			xmlRPC: NewXmlRPC() }
}

func (s* Supervisor) GetConfig() *Config {
	return s.config
}

func (s* Supervisor ) GetVersion(r *http.Request, args *struct { }, reply *struct{ Version string} ) error {
	reply.Version = VERSION
	return nil
}

func (s *Supervisor) GetSupervisorVersion( r *http.Request, args *struct { }, reply *struct{ Version string}  ) error {
	reply.Version = VERSION
	return nil
}

func (s *Supervisor) GetIdentification( r *http.Request, args *struct { }, reply *struct{ Id string}  ) error {
	entry, ok := s.config.GetSupervisord()
	if ok {
		reply.Id = entry.GetString( "identifier", "No-supervisorId-set")
	} else {
		reply.Id = "No-supervisorId-set"
	}
	return nil
}

func (s *Supervisor) GetState( r *http.Request, args *struct { }, reply *processStateInfo ) error {
	//statecode     statename
	//=======================
	// 2            FATAL
	// 1            RUNNING
	// 0            RESTARTING
	// -1           SHUTDOWN
	reply.Statecode = 1
	reply.Statename = "RUNNING"
	return nil
}

func (s *Supervisor) GetPID( r *http.Request, args *struct { }, reply *struct{ Pid int } ) error {
	reply.Pid = os.Getpid()
	return nil
}

func (s *Supervisor) ReadLog( r *http.Request, args *logReadInfo, reply *struct { Log string } ) error {
	reply.Log = "not implemented"
	return nil
}

func (s* Supervisor) Shutdown( r *http.Request, args *struct { }, reply *struct{ Ret bool } ) error {
	reply.Ret = true
	return nil
}

func (s* Supervisor) Restart( r *http.Request, args *struct { }, reply *struct{ Ret bool } ) error {
	reply.Ret = true
	return nil
}

func getProcessInfo( proc *Process) *processInfo {
	return &processInfo { Name: proc.GetName(),
                                Group: proc.GetGroup(),
                                Description: proc.GetDescription(),
                                Start: proc.GetStartTime().Unix(),
                                Stop: proc.GetStopTime().Unix(),
                                Now: time.Now().Unix(),
                                State: int( proc.GetState() ),
                                Statename: proc.GetState().String(),
                                Spawnerr: "",
                                Exitstatus: proc.GetExitstatus(),
                                Logfile: proc.GetStdoutLogfile(),
                                Stdout_logfile: proc.GetStdoutLogfile(),
                                Stderr_logfile: proc.GetStderrLogfile(),
                                Pid: proc.GetPid() }

}


func (s *Supervisor) GetAllProcessInfo( r *http.Request, args *struct { }, reply *struct{ AllProcessInfo []processInfo} ) error {
	reply.AllProcessInfo = make([]processInfo, 0 )
	s.procMgr.ForEachProcess( func (proc *Process) {
		procInfo := getProcessInfo( proc )
		reply.AllProcessInfo = append(  reply.AllProcessInfo, *procInfo )
	} )

	return nil
}

func (s *Supervisor) GetProcessInfo( r *http.Request, args *struct { name string }, reply *struct{procInfo processInfo} ) error {
	proc := s.procMgr.Find( args.name )
	if proc == nil {
		return fmt.Errorf( "no process named %s", args.name )
	}

	reply.procInfo = *getProcessInfo( proc )
	return nil
}

func (s *Supervisor) StartProcess( r* http.Request, args* StartProcessArgs, reply *struct{ Success bool } ) error {
	proc := s.procMgr.Find( args.Name )

	if proc == nil {
		return fmt.Errorf( "fail to find process %s", args.Name )
	}
	proc.Start()
	reply.Success = true
	return nil
}

func (s *Supervisor) StartAllProcesses( r* http.Request, args* struct { Wait bool `default:"true"`}, reply *struct{ AllProcessInfo []processInfo } ) error {
	s.procMgr.ForEachProcess( func( proc *Process ) {
		proc.Start()
	})
	return nil
}

func (s *Supervisor) StartProcessGroup( r* http.Request, args* StartProcessArgs,  reply *struct{ AllProcessInfo []processInfo } ) error {
	s.procMgr.ForEachProcess( func( proc *Process) {
		if proc.GetGroup() == args.Name {
			proc.Start()
		}
	})
        return nil
}


func (s *Supervisor) StopProcess( r* http.Request, args* StartProcessArgs, reply *struct{ Success bool } ) error {
	proc := s.procMgr.Find( args.Name )
	if proc == nil {
		return fmt.Errorf( "fail to find process %s", args.Name )
	}
	proc.Stop()
	reply.Success = true
	return nil
}

func (s *Supervisor) StopProcessGroup( r* http.Request, args* StartProcessArgs,  reply *struct{ AllProcessInfo []processInfo } ) error {
	 s.procMgr.ForEachProcess( func( proc *Process) {
                if proc.GetGroup() == args.Name {
                        proc.Stop()
                }
        })
        return nil
}

func (s *Supervisor) StopAllProcesses( r* http.Request, args* struct { Wait bool `default:"true"`}, reply *struct{ AllProcessInfo []processInfo } ) error {
	s.procMgr.ForEachProcess( func( proc *Process) {
		proc.Stop()
	})
        return nil
}

func (s *Supervisor) SignalProcess( r* http.Request, args* processSignal, reply *struct{ Success bool } ) error {
	proc := s.procMgr.Find( args.Name )
	if proc == nil {
		return fmt.Errorf( "No process named %s", args.Name )
	}
	proc.Signal( toSignal(args.Signal) )
        return nil
}

func (s *Supervisor) SignalProcessGroup( r* http.Request, args* processSignal, reply *struct{  AllProcessInfo []processInfo } ) error {
	s.procMgr.ForEachProcess( func( proc *Process ) {
		if proc.GetGroup() == args.Name {
			proc.Signal( toSignal( args.Signal ) )
		}
	})
        return nil
}

func (s *Supervisor) SignalAllProcesses( r* http.Request, args* processSignal, reply *struct{  AllProcessInfo []processInfo } ) error {
	s.procMgr.ForEachProcess( func( proc *Process ) {
		proc.Signal( toSignal( args.Signal ) )
	})
        return nil
}

func (s *Supervisor) SendProcessStdin( r* http.Request, args* processStdin, reply *struct{  Success bool } ) error {
	proc := s.procMgr.Find( args.Name )
	if proc != nil || proc.GetState() != RUNNING {
		return fmt.Errorf( "NOT_RUNNING" )
	}
	return proc.SendProcessStdin( args.Chars )
}

func (s *Supervisor) SendRemoteCommEvent(  r* http.Request, args* remoteCommEvent, reply *struct{  Success bool } ) error {
	return fmt.Errorf( "Not implemented" )
}

func (s *Supervisor) Reload() error {
	//get the previous loaded programs
	prevPrograms := s.config.GetProgramNames()

	err := s.config.Load()

        if err == nil {

		programs := s.config.GetProgramNames()
		for _, entry := range s.config.GetPrograms() {
			s.procMgr.CreateProcess( entry )
		}
		removedPrograms := sub( prevPrograms, programs )
		for _, p := range( removedPrograms ) {
			s.procMgr.Remove( p )
		}
		httpServerConfig, ok := s.config.GetInetHttpServer()
		if ok {
			addr := httpServerConfig.GetString( "port", "" )
			if addr != "" {
				s.xmlRPC.Stop()
				s.xmlRPC.Start( addr, s )
			}
		}
	}
	return err

}

func sub( arr_1 []string, arr_2 []string) []string {
	result := make([]string,0)
	for _, s := range arr_1 {
		exist := false
		for _, s2 := range arr_2 {
			if s == s2 {
				exist = true
			}
		}
		if ! exist {
			result = append( result, s )
		}
	}
	return result
}

func (s *Supervisor) ReloadConfig (  r* http.Request, args* struct { }, reply *struct{  Success bool } ) error {
	err := s.Reload()
	if err == nil {
		reply.Success = true
	} else {
		reply.Success = false
	}
	return err
}

func (s *Supervisor) AddProcessGroup(  r* http.Request, args* struct { Name string }, reply *struct{  Success bool } ) error {
	reply.Success  = false
        return nil
}

func (s *Supervisor) RemoveProcessGroup(  r* http.Request, args* struct { Name string }, reply *struct{  Success bool }  ) error {
	reply.Success = false
	return nil
}

