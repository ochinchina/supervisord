// +build !linux
// +build !windows

package process

import (
	"syscall"
)

func set_deathsig(sysProcAttr *syscall.SysProcAttr) {
	sysProcAttr.Setpgid = true
}
