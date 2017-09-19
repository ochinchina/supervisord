// +build !linux
// +build !windows

package main

import (
	"syscall"
)

func set_deathsig(sysProcAttr *syscall.SysProcAttr) {
	sysProcAttr.Setpgid = true
}
