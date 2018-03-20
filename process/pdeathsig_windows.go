// +build windows

package process

import (
	"syscall"
)

func set_deathsig(_ *syscall.SysProcAttr) {
}
