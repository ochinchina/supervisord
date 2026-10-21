//go:build windows
// +build windows

package main

import (
	"os"

	"golang.org/x/sys/windows"
)

func enableConsoleVT() {
	var mode uint32
	handle := windows.Handle(os.Stdout.Fd())
	if err := windows.GetConsoleMode(handle, &mode); err == nil {
		err = windows.SetConsoleMode(handle, mode|windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING)
	}
}
