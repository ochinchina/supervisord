// +build !windows

package main

import (
	daemon "github.com/sevlyar/go-daemon"
	log "github.com/sirupsen/logrus"
)

func Deamonize(proc func()) {
	context := new(daemon.Context)

	child, err := context.Reborn()
	if err != nil {
		log.WithFields(log.Fields{"err": err}).Fatal("Unable to run")
	}
	if child != nil {
		return
	}
	defer context.Release()

	log.Info("daemon started")

	proc()
}
