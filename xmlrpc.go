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
	xmlrpcCodec.RegisterAlias( "supervisor.getSupervisorVersion", "Supervisor.GetVersion" )
	xmlrpcCodec.RegisterAlias( "supervisor.getAllProcessInfo", "Supervisor.GetAllProcessInfo" )
	xmlrpcCodec.RegisterAlias( "supervisor.startProcess", "Supervisor.StartProcess" )

	http.Handle("/RPC2", RPC)
	var err error
	p.listener, err = net.Listen( "tcp", listenAddr )
	if err == nil {
		http.Serve(p.listener, nil )
	}
}
