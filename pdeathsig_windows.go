// +build windows
package main

import (
	"syscall"
)

func set_deathsig(_ *syscall.SysProcAttr) {
}
