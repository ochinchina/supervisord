// +build !windows

package process

import (
	"syscall"
)

func setUserID(procAttr *syscall.SysProcAttr, uid uint32, gid uint32) {
	procAttr.Credential = &syscall.Credential{Uid: uid, Gid: gid, NoSetGroups: true}
}
