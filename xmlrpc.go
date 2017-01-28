package main

import (
	"net"
	"net/http"
	"github.com/gorilla/rpc"
	"github.com/ochinchina/gorilla-xmlrpc/xml"
)

type XmlRPC struct {
	unixListener net.Listener
	inetListener net.Listener
}


func NewXmlRPC() *XmlRPC {
	return &XmlRPC{}
}

func (p *XmlRPC) Stop() {
	if p.unixListener != nil {
		p.unixListener.Close()
	}
	if p.inetListener != nil {
		p.inetListener.Close()
	}
}

func (p *XmlRPC) StartUnixHttpServer( listenAddr string, s *Supervisor ) {
	mux := http.NewServeMux()
	mux.Handle("/RPC2", p.createRPCServer(s) )
	var err error
	p.unixListener, err = net.Listen( "unix", listenAddr )
	if err == nil {
		http.Serve(p.unixListener, mux )
	}


}

func (p *XmlRPC) StartInetHttpServer( listenAddr string, s *Supervisor )  {
	mux := http.NewServeMux()
	mux.Handle("/RPC2", p.createRPCServer(s) )
	var err error
	p.inetListener, err = net.Listen( "tcp", listenAddr )
	if err == nil {
		http.Serve(p.inetListener, mux )
	}
}

func (p *XmlRPC) createRPCServer( s* Supervisor ) *rpc.Server {
	RPC := rpc.NewServer()
        xmlrpcCodec := xml.NewCodec()
        RPC.RegisterCodec(xmlrpcCodec, "text/xml")
        RPC.RegisterService( s, "" )

        xmlrpcCodec.RegisterAlias( "supervisor.getVersion", "Supervisor.GetVersion" )
        xmlrpcCodec.RegisterAlias( "supervisor.getAPIVersion", "Supervisor.GetVersion" )
        xmlrpcCodec.RegisterAlias( "supervisor.getIdentification", "Supervisor.GetIdentification" )
        xmlrpcCodec.RegisterAlias( "supervisor.getState", "Supervisor.GetState" )
        xmlrpcCodec.RegisterAlias( "supervisor.getPID", "Supervisor.GetPID" )
        xmlrpcCodec.RegisterAlias( "supervisor.readLog", "Supervisor.ReadLog" )
        xmlrpcCodec.RegisterAlias( "supervisor.clearLog", "Supervisor.ClearLog" )
        xmlrpcCodec.RegisterAlias( "supervisor.shutdown", "Supervisor.Shutdown" )
        xmlrpcCodec.RegisterAlias( "supervisor.restart", "Supervisor.Restart" )
        xmlrpcCodec.RegisterAlias( "supervisor.getProcessInfo", "Supervisor.GetProcessInfo" )
        xmlrpcCodec.RegisterAlias( "supervisor.getSupervisorVersion", "Supervisor.GetVersion" )
        xmlrpcCodec.RegisterAlias( "supervisor.getAllProcessInfo", "Supervisor.GetAllProcessInfo" )
        xmlrpcCodec.RegisterAlias( "supervisor.startProcess", "Supervisor.StartProcess" )
        xmlrpcCodec.RegisterAlias( "supervisor.startAllProcesses", "Supervisor.StartAllProcesses" )
        xmlrpcCodec.RegisterAlias( "supervisor.startProcessGroup", "Supervisor.StartProcessGroup" )
        xmlrpcCodec.RegisterAlias( "supervisor.stopProcess", "Supervisor.StopProcess" )
        xmlrpcCodec.RegisterAlias( "supervisor.stopProcessGroup", "Supervisor.StopProcessGroup" )
        xmlrpcCodec.RegisterAlias( "supervisor.stopAllProcesses", "Supervisor.StopAllProcesses" )
        xmlrpcCodec.RegisterAlias( "supervisor.signalProcess", "Supervisor.SignalProcess" )
        xmlrpcCodec.RegisterAlias( "supervisor.signalProcessGroup", "Supervisor.SignalProcessGroup" )
        xmlrpcCodec.RegisterAlias( "supervisor.signalAllProcesses", "Supervisor.SignalAllProcesses" )
        xmlrpcCodec.RegisterAlias( "supervisor.sendProcessStdin", "Supervisor.SendProcessStdin" )
        xmlrpcCodec.RegisterAlias( "supervisor.sendRemoteCommEvent", "Supervisor.SendRemoteCommEvent" )
        xmlrpcCodec.RegisterAlias( "supervisor.reloadConfig", "Supervisor.Reload" )
        xmlrpcCodec.RegisterAlias( "supervisor.addProcessGroup", "Supervisor.AddProcessGroup" )
        xmlrpcCodec.RegisterAlias( "supervisor.removeProcessGroup", "Supervisor.RemoveProcessGroup" )
        xmlrpcCodec.RegisterAlias( "supervisor.readProcessStdoutLog", "Supervisor.ReadProcessStdoutLog" )
        xmlrpcCodec.RegisterAlias( "supervisor.readProcessStderrLog", "Supervisor.ReadProcessStderrLog" )
        xmlrpcCodec.RegisterAlias( "supervisor.tailProcessStdoutLog", "Supervisor.TailProcessStdoutLog" )
        xmlrpcCodec.RegisterAlias( "supervisor.tailProcessStderrLog", "Supervisor.TailProcessStderrLog" )
        xmlrpcCodec.RegisterAlias( "supervisor.clearProcessLogs", "Supervisor.ClearProcessLogs" )
        xmlrpcCodec.RegisterAlias( "supervisor.clearAllProcessLogs", "Supervisor.ClearAllProcessLogs" )
	return RPC
}
