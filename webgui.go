package main

import (
	"net/http"

	"github.com/gorilla/mux"
)

//SupervisorWebgui is interface to serve webgui
type SupervisorWebgui struct {
	router     *mux.Router
	supervisor *Supervisor
}

//NewSupervisorWebgui returns object for SupervisorWebgui
func NewSupervisorWebgui(supervisor *Supervisor) *SupervisorWebgui {
	router := mux.NewRouter()
	return &SupervisorWebgui{router: router, supervisor: supervisor}
}

//CreateHandler handles Webgui requests
func (sw *SupervisorWebgui) CreateHandler() http.Handler {
	sw.router.PathPrefix("/").Handler(http.FileServer(HTTP))
	return sw.router
}
