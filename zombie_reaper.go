// +build !windows

package main

import (
	reaper "github.com/ochinchina/go-reaper"
)

// ReapZombie reap the zombie child process
func ReapZombie() {
	go reaper.Reap()
}
