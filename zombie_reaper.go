// +build !windows

package gopm

import (
	"github.com/ramr/go-reaper"
)

// ReapZombie reap the zombie child process
func ReapZombie() {
	go reaper.Reap()
}
