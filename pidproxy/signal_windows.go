//go:build windows
// +build windows

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
		syscall.SIGQUIT)
}

func allowForwardSig(_ os.Signal) bool {
	return true
}
