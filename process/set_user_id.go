//go:build !windows
// +build !windows

package process

import (
	log "github.com/sirupsen/logrus"
	"os/user"
	"strconv"
	"syscall"
)

func setUserID(procAttr *syscall.SysProcAttr, uid uint32, gid uint32) {

	u, err := user.Current()
	if err == nil {
		cuid, uid_err := strconv.ParseUint(u.Uid, 10, 32)
		cgid, gid_err := strconv.ParseUint(u.Gid, 10, 32)
		if uid_err == nil && gid_err == nil && uint32(cuid) == uid && uint32(cgid) == gid {
			log.Info("no need to switch user")
			return
		}
	}
	procAttr.Credential = &syscall.Credential{Uid: uid, Gid: gid, NoSetGroups: true}
}
