// +build windows

package main

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
)

func getSysProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{}
}

func kill(p *Process, sig os.Signal) error {
	return exec.Command("TASKKILL", "/F", "/T", "/PID", fmt.Sprint(p.cmd.Process.Pid)).Run()
}
