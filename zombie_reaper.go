// +build !windows

package main

import (
	reaper "github.com/ochinchina/go-reaper"
)

//ReapZombie reaps zombies if supervisor is PID1
func ReapZombie() {
	go reaper.Reap()
}
