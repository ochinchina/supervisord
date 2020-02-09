package main

import (
	"crypto/sha1"
	"encoding/hex"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"sync/atomic"

	"github.com/gorilla/rpc"
	"github.com/ochinchina/gorilla-xmlrpc/xml"
	log "github.com/sirupsen/logrus"
)

//XMLRPC is struct to hold XMLRPC processing
type XMLRPC struct {
	listeners map[string]net.Listener
	// true if RPC is started
	started *uint32
}

//HTTPBasicAuth struct holds user auth data and handler
type HTTPBasicAuth struct {
	user     string
	password string
	handler  http.Handler
}

//NewHTTPBasicAuth returns new HTTPBasicAuth object
func NewHTTPBasicAuth(user string, password string, handler http.Handler) *HTTPBasicAuth {
	if user != "" && password != "" {
		log.Debug("require authentication")
	}
	return &HTTPBasicAuth{user: user, password: password, handler: handler}
}

//ServeHTTP serves HTTP
func (h *HTTPBasicAuth) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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

//NewXMLRPC returns new XML RPC obj
func NewXMLRPC() *XMLRPC {
	p := XMLRPC{listeners: make(map[string]net.Listener)}
	atomic.AddUint32(p.started, 0)
	return &p
}

// Stop stop network listening
func (p *XMLRPC) Stop() {
	log.Info("stop listening")
	for _, listener := range p.listeners {
		listener.Close()
	}
	atomic.AddUint32(p.started, 0)
}

//StartUnixHTTPServer starts Unix HTTP Server
func (p *XMLRPC) StartUnixHTTPServer(user string, password string, listenAddr string, s *Supervisor) {
	os.Remove(listenAddr)
	p.startHTTPServer(user, password, "unix", listenAddr, s)
}

//StartInetHTTPServer starts INET HTTP Server
func (p *XMLRPC) StartInetHTTPServer(user string, password string, listenAddr string, s *Supervisor) {
	p.startHTTPServer(user, password, "tcp", listenAddr, s)
}

func (p *XMLRPC) startHTTPServer(user string, password string, protocol string, listenAddr string, s *Supervisor) {
	if atomic.CompareAndSwapUint32(p.started, 0, 1) == false /* swapped = false */ {
		return
	}
	mux := http.NewServeMux()
	mux.Handle("/RPC2", NewHTTPBasicAuth(user, password, p.createRPCServer(s)))
	progRESTHandler := NewSupervisorRestful(s).CreateProgramHandler()
	mux.Handle("/program/", NewHTTPBasicAuth(user, password, progRESTHandler))
	supervisorRESTHandler := NewSupervisorRestful(s).CreateSupervisorHandler()
	mux.Handle("/supervisor/", NewHTTPBasicAuth(user, password, supervisorRESTHandler))
	logtailHandler := NewLogtail(s).CreateHandler()
	mux.Handle("/logtail/", NewHTTPBasicAuth(user, password, logtailHandler))
	webGUIHandler := NewSupervisorWebgui(s).CreateHandler()
	mux.Handle("/", NewHTTPBasicAuth(user, password, webGUIHandler))
	listener, err := net.Listen(protocol, listenAddr)
	if err == nil {
		log.WithFields(log.Fields{"addr": listenAddr, "protocol": protocol}).Info("success to listen on address")
		p.listeners[protocol] = listener
		http.Serve(listener, mux)
	} else {
		log.WithFields(log.Fields{"addr": listenAddr, "protocol": protocol}).Fatal("fail to listen on address")
	}

}
func (p *XMLRPC) createRPCServer(s *Supervisor) *rpc.Server {
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
