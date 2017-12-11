// +build windows

package process

import (
	"syscall"
)

func set_user_id(_ *syscall.SysProcAttr, _ uint32, _ uint32) {

}
