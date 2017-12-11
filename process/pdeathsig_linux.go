// +build linux

package process

import (
	"syscall"
)

func set_deathsig(sysProcAttr *syscall.SysProcAttr) {
	sysProcAttr.Setpgid = true
	sysProcAttr.Pdeathsig = syscall.SIGKILL
}
