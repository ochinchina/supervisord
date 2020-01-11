package main

import (
	"crypto/sha1"
	"encoding/hex"
	"io"
	"net"
	"net/http"
	"os"
	"strings"

	"github.com/gorilla/rpc"
	"github.com/ochinchina/gorilla-xmlrpc/xml"
	log "github.com/sirupsen/logrus"
)

type XmlRPC struct {
	listeners map[string]net.Listener
	// true if RPC is started
	started bool
}

type httpBasicAuth struct {
	user     string
	password string
	handler  http.Handler
}

func NewHttpBasicAuth(user string, password string, handler http.Handler) *httpBasicAuth {
	if user != "" && password != "" {
		log.Debug("require authentication")
	}
	return &httpBasicAuth{user: user, password: password, handler: handler}
}

func (h *httpBasicAuth) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.user == "" || h.password == "" {
		log.Debug("no auth required")
		h.handler.ServeHTTP(w, r)
		return
	}
	username, password, ok := r.BasicAuth()
	if ok && username == h.user {
		if strings.HasPrefix(h.password, "{SHA}") {
			log.Debug("auth with SHA")
			hash := sha1.New()
			io.WriteString(hash, password)
			if hex.EncodeToString(hash.Sum(nil)) == h.password[5:] {
				h.handler.ServeHTTP(w, r)
				return
			}
		} else if password == h.password {
			log.Debug("Auth with normal password")
			h.handler.ServeHTTP(w, r)
			return
		}
	}
	w.Header().Set("WWW-Authenticate", "Basic realm=\"supervisor\"")
	w.WriteHeader(401)
}

func NewXmlRPC() *XmlRPC {
	return &XmlRPC{listeners: make(map[string]net.Listener), started: false}
}

// stop network listening
func (p *XmlRPC) Stop() {
	log.Info("stop listening")
	for _, listener := range p.listeners {
		listener.Close()
	}
	p.started = false
}

func (p *XmlRPC) StartUnixHttpServer(user string, password string, listenAddr string, s *Supervisor) {
	os.Remove(listenAddr)
	p.startHttpServer(user, password, "unix", listenAddr, s)
}

func (p *XmlRPC) StartInetHttpServer(user string, password string, listenAddr string, s *Supervisor) {
	p.startHttpServer(user, password, "tcp", listenAddr, s)
}

func (p *XmlRPC) startHttpServer(user string, password string, protocol string, listenAddr string, s *Supervisor) {
	if p.started {
		return
	}
	p.started = true
	mux := http.NewServeMux()
	mux.Handle("/RPC2", NewHttpBasicAuth(user, password, p.createRPCServer(s)))
	prog_rest_handler := NewSupervisorRestful(s).CreateProgramHandler()
	mux.Handle("/program/", NewHttpBasicAuth(user, password, prog_rest_handler))
	supervisor_rest_handler := NewSupervisorRestful(s).CreateSupervisorHandler()
	mux.Handle("/supervisor/", NewHttpBasicAuth(user, password, supervisor_rest_handler))
	logtail_handler := NewLogtail(s).CreateHandler()
	mux.Handle("/logtail/", NewHttpBasicAuth(user, password, logtail_handler))
	webgui_handler := NewSupervisorWebgui(s).CreateHandler()
	mux.Handle("/", NewHttpBasicAuth(user, password, webgui_handler))
	listener, err := net.Listen(protocol, listenAddr)
	if err == nil {
		log.WithFields(log.Fields{"addr": listenAddr, "protocol": protocol}).Info("success to listen on address")
		p.listeners[protocol] = listener
		http.Serve(listener, mux)
	} else {
		log.WithFields(log.Fields{"addr": listenAddr, "protocol": protocol}).Fatal("fail to listen on address")
	}

}
func (p *XmlRPC) createRPCServer(s *Supervisor) *rpc.Server {
	RPC := rpc.NewServer()
	xmlrpcCodec := xml.NewCodec()
	RPC.RegisterCodec(xmlrpcCodec, "text/xml")
	RPC.RegisterService(s, "")

	xmlrpcCodec.RegisterAlias("supervisor.getVersion", "Supervisor.GetVersion")
	xmlrpcCodec.RegisterAlias("supervisor.getAPIVersion", "Supervisor.GetVersion")
	xmlrpcCodec.RegisterAlias("supervisor.getIdentification", "Supervisor.GetIdentification")
	xmlrpcCodec.RegisterAlias("supervisor.getState", "Supervisor.GetState")
	xmlrpcCodec.RegisterAlias("supervisor.getPID", "Supervisor.GetPID")
	xmlrpcCodec.RegisterAlias("supervisor.readLog", "Supervisor.ReadLog")
	xmlrpcCodec.RegisterAlias("supervisor.clearLog", "Supervisor.ClearLog")
	xmlrpcCodec.RegisterAlias("supervisor.shutdown", "Supervisor.Shutdown")
	xmlrpcCodec.RegisterAlias("supervisor.restart", "Supervisor.Restart")
	xmlrpcCodec.RegisterAlias("supervisor.getProcessInfo", "Supervisor.GetProcessInfo")
	xmlrpcCodec.RegisterAlias("supervisor.getSupervisorVersion", "Supervisor.GetVersion")
	xmlrpcCodec.RegisterAlias("supervisor.getAllProcessInfo", "Supervisor.GetAllProcessInfo")
	xmlrpcCodec.RegisterAlias("supervisor.startProcess", "Supervisor.StartProcess")
	xmlrpcCodec.RegisterAlias("supervisor.startAllProcesses", "Supervisor.StartAllProcesses")
	xmlrpcCodec.RegisterAlias("supervisor.startProcessGroup", "Supervisor.StartProcessGroup")
	xmlrpcCodec.RegisterAlias("supervisor.stopProcess", "Supervisor.StopProcess")
	xmlrpcCodec.RegisterAlias("supervisor.stopProcessGroup", "Supervisor.StopProcessGroup")
	xmlrpcCodec.RegisterAlias("supervisor.stopAllProcesses", "Supervisor.StopAllProcesses")
	xmlrpcCodec.RegisterAlias("supervisor.signalProcess", "Supervisor.SignalProcess")
	xmlrpcCodec.RegisterAlias("supervisor.signalProcessGroup", "Supervisor.SignalProcessGroup")
	xmlrpcCodec.RegisterAlias("supervisor.signalAllProcesses", "Supervisor.SignalAllProcesses")
	xmlrpcCodec.RegisterAlias("supervisor.sendProcessStdin", "Supervisor.SendProcessStdin")
	xmlrpcCodec.RegisterAlias("supervisor.sendRemoteCommEvent", "Supervisor.SendRemoteCommEvent")
	xmlrpcCodec.RegisterAlias("supervisor.reloadConfig", "Supervisor.ReloadConfig")
	xmlrpcCodec.RegisterAlias("supervisor.addProcessGroup", "Supervisor.AddProcessGroup")
	xmlrpcCodec.RegisterAlias("supervisor.removeProcessGroup", "Supervisor.RemoveProcessGroup")
	xmlrpcCodec.RegisterAlias("supervisor.readProcessStdoutLog", "Supervisor.ReadProcessStdoutLog")
	xmlrpcCodec.RegisterAlias("supervisor.readProcessStderrLog", "Supervisor.ReadProcessStderrLog")
	xmlrpcCodec.RegisterAlias("supervisor.tailProcessStdoutLog", "Supervisor.TailProcessStdoutLog")
	xmlrpcCodec.RegisterAlias("supervisor.tailProcessStderrLog", "Supervisor.TailProcessStderrLog")
	xmlrpcCodec.RegisterAlias("supervisor.clearProcessLogs", "Supervisor.ClearProcessLogs")
	xmlrpcCodec.RegisterAlias("supervisor.clearAllProcessLogs", "Supervisor.ClearAllProcessLogs")
	return RPC
}
