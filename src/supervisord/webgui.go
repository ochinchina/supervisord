package main

import (
	"net/http"

	"github.com/gorilla/mux"
)

// SupervisorWebgui the interface to show a WEBGUI to control the supervisor
type SupervisorWebgui struct {
	router     *mux.Router
	supervisor *Supervisor
}

// NewSupervisorWebgui create a new SupervisorWebgui object
func NewSupervisorWebgui(supervisor *Supervisor) *SupervisorWebgui {
	router := mux.NewRouter()
	return &SupervisorWebgui{router: router, supervisor: supervisor}
}

// CreateHandler create a http handler to process the request from WEBGUI
func (sw *SupervisorWebgui) CreateHandler() http.Handler {
	sw.router.PathPrefix("/").Handler(http.FileServer(HTTP))
	return sw.router
}
