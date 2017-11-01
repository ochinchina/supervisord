// +build !windows

package main

import (
	"os"
	"syscall"
)

func getSysProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{Setpgid: true}
}

func kill(p *Process, sig os.Signal) error {
	localSig := sig.(syscall.Signal)
	return syscall.Kill(-p.cmd.Process.Pid, localSig)
}
