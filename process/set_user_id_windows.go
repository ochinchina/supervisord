// +build windows

package process

import (
	"syscall"
)

func setUserID(_ *syscall.SysProcAttr, _ uint32, _ uint32) {

}
