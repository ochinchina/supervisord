//go:build linux
// +build linux

package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strconv"
	"syscall"
	"time"
)

// reaperGrace is how long a child must stay a zombie before the reaper
// collects it. Every child supervisord starts itself has a goroutine blocked
// in exec.Cmd.Wait that collects it as soon as it exits, so a zombie that
// outlives the grace period is an orphan nobody waits for.
const reaperGrace = time.Second

// ReapZombie reaps orphaned zombies when supervisord runs as PID 1.
//
// It never waits for any child (wait4(-1)): doing so steals the exit status
// of the programs supervisord supervises, so exec.Cmd.Wait fails with ECHILD,
// the exit code is lost and a program with autorestart=unexpected is not
// restarted.
func ReapZombie() {
	if os.Getpid() != 1 {
		return
	}
	go reapOrphans(nil, reaperGrace)
}

// zombie identifies a zombie child; the start time guards against pid reuse.
type zombie struct {
	pid       int
	startTime string
}

// reapOrphans collects zombie children of this process that have been zombies
// for at least grace, until stop is closed.
func reapOrphans(stop <-chan struct{}, grace time.Duration) {
	ticker := time.NewTicker(grace / 2)
	defer ticker.Stop()
	seen := make(map[zombie]time.Time)
	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
		}
		seen = reapOnce(os.Getpid(), grace, seen, time.Now())
	}
}

// reapOnce scans /proc for zombie children of ppid, reaps those first seen at
// least grace before now and returns the zombies still being watched.
func reapOnce(ppid int, grace time.Duration, seen map[zombie]time.Time, now time.Time) map[zombie]time.Time {
	next := make(map[zombie]time.Time)
	for _, z := range zombieChildren(ppid) {
		first, ok := seen[z]
		if !ok {
			next[z] = now
			continue
		}
		if now.Sub(first) < grace {
			next[z] = first
			continue
		}
		var ws syscall.WaitStatus
		if _, err := syscall.Wait4(z.pid, &ws, syscall.WNOHANG, nil); err == syscall.EINTR {
			next[z] = first
		}
	}
	return next
}

// zombieChildren lists the children of ppid that are zombies.
func zombieChildren(ppid int) []zombie {
	stats, _ := filepath.Glob("/proc/[0-9]*/stat")
	var zombies []zombie
	for _, stat := range stats {
		b, err := os.ReadFile(stat)
		if err != nil {
			continue // the process is gone
		}
		// pid (comm) state ppid ... with starttime the 22nd field; comm may
		// contain spaces and parentheses, so split after the last ')'
		i := bytes.LastIndexByte(b, ')')
		if i < 0 {
			continue
		}
		fields := bytes.Fields(b[i+1:])
		if len(fields) < 20 || string(fields[0]) != "Z" || string(fields[1]) != strconv.Itoa(ppid) {
			continue
		}
		pid, err := strconv.Atoi(filepath.Base(filepath.Dir(stat)))
		if err != nil {
			continue
		}
		zombies = append(zombies, zombie{pid: pid, startTime: string(fields[19])})
	}
	return zombies
}
