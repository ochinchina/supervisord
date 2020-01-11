// +build !windows

package main

import (
	daemon "github.com/ochinchina/go-daemon"
	log "github.com/sirupsen/logrus"
)

func Deamonize(proc func()) {
	context := daemon.Context{LogFileName: "/dev/stdout"}

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
