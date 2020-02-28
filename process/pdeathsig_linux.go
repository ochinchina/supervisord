// +build linux

package process

import (
	"syscall"
)

func setDeathsig(sysProcAttr *syscall.SysProcAttr) {
	sysProcAttr.Setpgid = true
	sysProcAttr.Pdeathsig = syscall.SIGKILL
}
