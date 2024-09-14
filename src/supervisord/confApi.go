package main

import (
	"net/http"

	"github.com/gorilla/mux"
)

type ConfApi struct {
	router     *mux.Router
	supervisor *Supervisor
}

// NewLogtail creates a Logtail object
func NewConfApi(supervisor *Supervisor) *ConfApi {
	return &ConfApi{router: mux.NewRouter(), supervisor: supervisor}
}

// CreateHandler creates http handlers to process the program stdout and stderr through http interface
func (ca *ConfApi) CreateHandler() http.Handler {
	ca.router.HandleFunc("/conf/{program}", ca.getProgramConfFile).Methods("GET")
	return ca.router
}

func (ca *ConfApi) getProgramConfFile(writer http.ResponseWriter, request *http.Request) {
	vars := mux.Vars(request)
	if vars == nil {
		writer.WriteHeader(http.StatusNotFound)
		return
	}

	programName := vars["program"]
	programConfigPath := getProgramConfigPath(programName, ca.supervisor)
	if programConfigPath == "" {
		writer.WriteHeader(http.StatusNotFound)
		return
	}

	b, err := readFile(programConfigPath)
	if err != nil {
		writer.WriteHeader(http.StatusNotFound)
		return
	}

	writer.WriteHeader(http.StatusOK)
	writer.Write(b)
}
