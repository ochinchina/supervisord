package main

import (
	"net/http"
	"github.com/gorilla/rpc"
	"github.com/ochinchina/gorilla-xmlrpc/xml"
)

type XmlRPC struct {
}

func NewXmlRPC() *XmlRPC {
	return &XmlRPC{}
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

	http.ListenAndServe(listenAddr, nil)
}
