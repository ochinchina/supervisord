// +build !windows

package main
import (
	reaper "github.com/ochinchina/go-reaper"
)

func ReapZombie() {
	go reaper.Reap()
}
