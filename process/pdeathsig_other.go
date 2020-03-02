// +build !linux
// +build !windows

package process

import (
	"syscall"
)

func setDeathsig(sysProcAttr *syscall.SysProcAttr) {
	sysProcAttr.Setpgid = true
}
