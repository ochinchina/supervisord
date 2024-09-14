//go:build !linux && !windows
// +build !linux,!windows

package process

import (
	"syscall"
)

func setDeathsig(sysProcAttr *syscall.SysProcAttr) {
	sysProcAttr.Setpgid = true
}
