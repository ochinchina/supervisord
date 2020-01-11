package main

import (
	"net/http"

	"github.com/gorilla/mux"
)

type SupervisorWebgui struct {
	router     *mux.Router
	supervisor *Supervisor
}

func NewSupervisorWebgui(supervisor *Supervisor) *SupervisorWebgui {
	router := mux.NewRouter()
	return &SupervisorWebgui{router: router, supervisor: supervisor}
}

func (sw *SupervisorWebgui) CreateHandler() http.Handler {
	sw.router.PathPrefix("/").Handler(http.FileServer(HTTP))
	return sw.router
}
