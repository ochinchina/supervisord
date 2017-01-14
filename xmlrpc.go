package main

import (
	"net"
	"net/http"
	"github.com/gorilla/rpc"
	"github.com/ochinchina/gorilla-xmlrpc/xml"
)

type XmlRPC struct {
	listener net.Listener
}


func NewXmlRPC() *XmlRPC {
	return &XmlRPC{}
}

func (p *XmlRPC) Stop() {
	if p.listener != nil {
		p.listener.Close()
	}
}

func (p *XmlRPC) Start( listenAddr string, s *Supervisor )  {
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

	http.Handle("/RPC2", RPC)
	var err error
	p.listener, err = net.Listen( "tcp", listenAddr )
	if err == nil {
		http.Serve(p.listener, nil )
	}
}
