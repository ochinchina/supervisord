// +build windows

package process

import (
	"syscall"
)

func setUserID(_ *syscall.SysProcAttr, _, _ uint32) {
}
