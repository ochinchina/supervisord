package main

import (
	"encoding/json"
	"github.com/gorilla/mux"
	"github.com/ochinchina/supervisord/types"
	"net/http"
)

type SupervisorRestful struct {
	router     *mux.Router
	supervisor *Supervisor
}

func NewSupervisorRestful(supervisor *Supervisor) *SupervisorRestful {
	router := mux.NewRouter()
	return &SupervisorRestful{router: router, supervisor: supervisor}
}

func (sr *SupervisorRestful) CreateHandler() http.Handler {
	sr.router.HandleFunc("/program/list", sr.ListProgram).Methods("GET")
	sr.router.HandleFunc("/program/start/{name}", sr.StartProgram).Methods("POST", "PUT")
	sr.router.HandleFunc("/program/stop/{name}", sr.StopProgram).Methods("POST", "PUT")
	sr.router.HandleFunc("/program/log/{name}/stdout", sr.ReadStdoutLog).Methods("GET")
	return sr.router
}

// list the status of all the programs
//
// json array to present the status of all programs
func (sr *SupervisorRestful) ListProgram(w http.ResponseWriter, req *http.Request) {
	result := struct{ AllProcessInfo []types.ProcessInfo }{make([]types.ProcessInfo, 0)}
	if sr.supervisor.GetAllProcessInfo(nil, nil, &result) == nil {
		json.NewEncoder(w).Encode(result.AllProcessInfo)
	} else {

	}
}

func (sr *SupervisorRestful) StartProgram(w http.ResponseWriter, req *http.Request) {
	params := mux.Vars(req)
	startArgs := StartProcessArgs{Name: params["name"], Wait: true}
	result := struct{ Success bool }{false}
	err := sr.supervisor.StartProcess(nil, &startArgs, &result)
	r := map[string]bool{"success": err == nil && result.Success}
	json.NewEncoder(w).Encode(&r)
}

func (sr *SupervisorRestful) StopProgram(w http.ResponseWriter, req *http.Request) {
	params := mux.Vars(req)
	stopArgs := StartProcessArgs{Name: params["name"], Wait: true}
	result := struct{ Success bool }{false}
	err := sr.supervisor.StopProcess(nil, &stopArgs, &result)
	r := map[string]bool{"success": err == nil && result.Success}
	json.NewEncoder(w).Encode(&r)
}

func (sr *SupervisorRestful) ReadStdoutLog(w http.ResponseWriter, req *http.Request) {
}
