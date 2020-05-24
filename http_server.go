package gopm

import (
	"net"
	"net/http"

	"go.uber.org/zap"
)

type HTTPServer struct {
	ln net.Listener
}

// Stop stop network listening
func (p *HTTPServer) Stop() {
	zap.L().Info("Stopping HTTP server")
	if p.ln != nil {
		_ = p.ln.Close()
		p.ln = nil
	}
}

// StartInetHTTPServer start http server on tcp with path listenAddr. If both user and password are not empty, the user
// must provide user and password for basic authentication when making a XML RPC request.
func (p *HTTPServer) Start(user, password, listenAddr string, s *Supervisor, startedCb func()) {
	if p.ln != nil {
		startedCb()
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
	listener, err := net.Listen("tcp", listenAddr)
	if err == nil {
		zap.L().Info("Start http", zap.String("addr", listenAddr))
		p.ln = listener
		startedCb()
		_ = http.Serve(listener, mux)
	} else {
		startedCb()
		zap.L().Fatal("Failed to listen on address", zap.Error(err), zap.String("addr", listenAddr))
	}
}
