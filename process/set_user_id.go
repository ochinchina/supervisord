// +build !windows

package process

import (
	"syscall"
)

func setUserID(procAttr *syscall.SysProcAttr, uid, gid uint32) {
	procAttr.Credential = &syscall.Credential{Uid: uid, Gid: gid, NoSetGroups: true}
}
