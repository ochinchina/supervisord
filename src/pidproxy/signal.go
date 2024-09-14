// +build !windows

package main

import (
	"os"
	"os/signal"
	"syscall"
)

func installSignal(c chan os.Signal) {
	signal.Notify(c, syscall.SIGTERM,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGUSR1,
		syscall.SIGUSR2,
		syscall.SIGQUIT,
		syscall.SIGCHLD)
}

func allowForwardSig(sig os.Signal) bool {
	return sig != syscall.SIGCHLD
}
