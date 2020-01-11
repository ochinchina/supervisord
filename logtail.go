package main

import (
	"github.com/gorilla/mux"
	logger "github.com/ochinchina/supervisord/logger"
	"net/http"
)

type Logtail struct {
	router     *mux.Router
	supervisor *Supervisor
}

func NewLogtail(supervisor *Supervisor) *Logtail {
	return &Logtail{router: mux.NewRouter(), supervisor: supervisor}
}

func (lt *Logtail) CreateHandler() http.Handler {
	lt.router.HandleFunc("/logtail/{program}/stdout", lt.getStdoutLog).Methods("GET")
	lt.router.HandleFunc("/logtail/{program}/stderr", lt.getStderrLog).Methods("GET")
	return lt.router
}

func (lt *Logtail) getStdoutLog(w http.ResponseWriter, req *http.Request) {
	lt.getLog("stdout", w, req)
}

func (lt *Logtail) getStderrLog(w http.ResponseWriter, req *http.Request) {
	lt.getLog("stderr", w, req)
}

func (lt *Logtail) getLog(logType string, w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	program := vars["program"]
	procMgr := lt.supervisor.GetProcessManager()
	proc := procMgr.Find(program)
	if proc == nil {
		w.WriteHeader(http.StatusBadRequest)
	} else {
		var ok bool = false
		var compositeLogger *logger.CompositeLogger = nil
		if logType == "stdout" {
			compositeLogger, ok = proc.StdoutLog.(*logger.CompositeLogger)
		} else {
			compositeLogger, ok = proc.StderrLog.(*logger.CompositeLogger)
		}
		if ok {
			w.Header().Set("Transfer-Encoding", "chunked")
			w.WriteHeader(http.StatusOK)
			flusher, _ := w.(http.Flusher)
			ch := make(chan []byte, 100)
			chanLogger := logger.NewChanLogger(ch)
			compositeLogger.AddLogger(chanLogger)
			for {
				text, ok := <-ch
				if !ok {
					break
				}
				_, err := w.Write(text)
				if err != nil {
					break
				}
				flusher.Flush()
			}
			compositeLogger.RemoveLogger(chanLogger)
			chanLogger.Close()
		}
	}

}
