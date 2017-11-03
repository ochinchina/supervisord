// +build windows

package main

import (
	"errors"
	log "github.com/Sirupsen/logrus"
	"os"
	"syscall"
)

//convert a signal name to signal
func toSignal(signalName string) (os.Signal, error) {
	if signalName == "HUP" {
		return syscall.SIGHUP, nil
	} else if signalName == "INT" {
		return syscall.SIGINT, nil
	} else if signalName == "QUIT" {
		return syscall.SIGQUIT, nil
	} else if signalName == "KILL" {
		return syscall.SIGKILL, nil
	} else if signalName == "USR1" {
		log.Warn("signal USR1 is not supported in windows")
		return nil, errors.New("signal USR1 is not supported in windows")
	} else if signalName == "USR2" {
		log.Warn("signal USR2 is not supported in windows")
		return nil, errors.New("signal USR2 is not supported in windows")
	} else {
		return syscall.SIGTERM, nil

	}

}

func kill(pid int, sig os.Signal) error {
	return errors.New("no Kill in windows")
}
