//go:build !windows
// +build !windows

package main

import (
	"github.com/ochinchina/go-daemon"
	log "github.com/sirupsen/logrus"
)

// Daemonize run this process in daemon mode
func Daemonize(logfile string, proc func()) {
	context := daemon.Context{LogFileName: logfile, PidFileName: "supervisord.pid"}

	child, err := context.Reborn()
	if err != nil {
		context := daemon.Context{}
		child, err = context.Reborn()
		if err != nil {
			log.WithFields(log.Fields{"err": err}).Fatal("Unable to run")
		}
	}
	if child != nil {
		return
	}
	defer context.Release()
	proc()
}
