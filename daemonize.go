// +build !windows

package main

import (
	"github.com/ochinchina/go-daemon"
	"go.uber.org/zap"
)

// Deamonize run this process in daemon mode
func Deamonize(proc func()) {
	context := daemon.Context{LogFileName: "/dev/stdout"}

	child, err := context.Reborn()
	if err != nil {
		context := daemon.Context{}
		child, err = context.Reborn()
		if err != nil {
			zap.L().Fatal("Unable to run", zap.Error(err))
		}
	}
	if child != nil {
		return
	}
	defer context.Release()
	proc()
}
