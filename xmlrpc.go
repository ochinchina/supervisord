package main

import (
	"crypto/sha1" //nolint:gosec
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"

	"github.com/gorilla/rpc"
	"github.com/ochinchina/gorilla-xmlrpc/xml"
	"github.com/ochinchina/supervisord/process"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
)

// XMLRPC mange the XML RPC servers
// start XML RPC servers to accept the XML RPC request from client side
type XMLRPC struct {
	// all the listeners to accept the XML RPC request
	listeners map[string]net.Listener
}

type httpBasicAuth struct {
	user     string
	password string
	handler  http.Handler
}

// create a new HttpBasicAuth object with username, password and the http request handler
func newHTTPBasicAuth(user string, password string, handler http.Handler) *httpBasicAuth {
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
			hash := sha1.New() //nolint:gosec
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

// NewXMLRPC create a new XML RPC object
func NewXMLRPC() *XMLRPC {
	return &XMLRPC{listeners: make(map[string]net.Listener)}
}

// Stop network listening
func (p *XMLRPC) Stop() {
	log.Info("stop listening")
	for _, listener := range p.listeners {
		listener.Close()
	}
	p.listeners = make(map[string]net.Listener)
}

// StartUnixHTTPServer start http server on unix domain socket with path listenAddr. If both user and password are not empty, the user
// must provide user and password for basic authentication when making an XML RPC request.
func (p *XMLRPC) StartUnixHTTPServer(user string, password string, listenAddr string, s *Supervisor, startedCb func()) {
	os.Remove(listenAddr)
	p.startHTTPServer(user, password, "unix", listenAddr, s, startedCb)
}

// StartInetHTTPServer start http server on tcp with path listenAddr. If both user and password are not empty, the user
// must provide user and password for basic authentication when making an XML RPC request.
func (p *XMLRPC) StartInetHTTPServer(user string, password string, listenAddr string, s *Supervisor, startedCb func()) {
	p.startHTTPServer(user, password, "tcp", listenAddr, s, startedCb)
}

func (p *XMLRPC) isHTTPServerStartedOnProtocol(protocol string) bool {
	_, ok := p.listeners[protocol]
	return ok
}

func (p *XMLRPC) startHTTPServer(user string, password string, protocol string, listenAddr string, s *Supervisor, startedCb func()) {
	if p.isHTTPServerStartedOnProtocol(protocol) {
		startedCb()
		return
	}

	conf := s.config
	m := conf.GetPrograms()
	for _, c := range m {
		fmt.Println("Name:", c.Name)
		//pro := s.procMgr.Find(val.Name)
		//cfg := pro.GetConfig()
		//realName := c.GetProgramName()
		res := c.GetString("stdout_logfile", "")
		//res := cfg.GetString(val.Name, "")
		fmt.Println("res:", res)
		fmt.Println("ConfigDir:", c.ConfigDir)
	}

	procCollector := process.NewProcCollector(s.procMgr)
	prometheus.Register(procCollector)
	mux := http.NewServeMux()
	mux.Handle("/RPC2", newHTTPBasicAuth(user, password, p.createRPCServer(s)))

	progRestHandler := NewSupervisorRestful(s).CreateProgramHandler()
	mux.Handle("/program/", newHTTPBasicAuth(user, password, progRestHandler))

	supervisorRestHandler := NewSupervisorRestful(s).CreateSupervisorHandler()
	mux.Handle("/supervisor/", newHTTPBasicAuth(user, password, supervisorRestHandler))

	logtailHandler := NewLogtail(s).CreateHandler()
	mux.Handle("/logtail/", newHTTPBasicAuth(user, password, logtailHandler))

	webguiHandler := NewSupervisorWebgui(s).CreateHandler()
	mux.Handle("/", newHTTPBasicAuth(user, password, webguiHandler))

	mux.Handle("/metrics", promhttp.Handler())

	listener, err := net.Listen(protocol, listenAddr)
	if err == nil {
		log.WithFields(log.Fields{"addr": listenAddr, "protocol": protocol}).Info("success to listen on address")
		p.listeners[protocol] = listener
		startedCb()
		http.Serve(listener, mux)
	} else {
		startedCb()
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
