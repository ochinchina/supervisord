//go:build linux
// +build linux

package process

import (
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"
	logtest "github.com/sirupsen/logrus/hooks/test"
)

func waitForState(t *testing.T, p *Process, want State, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for p.GetState() != want {
		if time.Now().After(deadline) {
			t.Fatalf("%s: state %v, want %v", p.GetName(), p.GetState(), want)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func exitStatus(p *Process) syscall.WaitStatus {
	p.lock.RLock()
	defer p.lock.RUnlock()
	if p.cmd == nil || p.cmd.ProcessState == nil {
		return 0
	}
	return p.cmd.ProcessState.Sys().(syscall.WaitStatus)
}

func forceKilled(hook *logtest.Hook, name string) bool {
	for _, e := range hook.AllEntries() {
		if e.Level == log.InfoLevel && e.Data["program"] == name && strings.HasPrefix(e.Message, "force to kill") {
			return true
		}
	}
	return false
}

// Stopping several programs at once, as StopAllProcesses does on shutdown:
// a program that exits quickly must not make the others get SIGKILL before
// their stopwaitsecs (ochinchina/supervisord#430). Each program must end by
// its own stop signal, and none may be reported as force-killed.
func TestStopAllGivesEachProgramItsStopwaitsecs(t *testing.T) {
	fast := newTestProcess(t, "fast", `[program:fast]
command=/bin/sh -c "exec sleep 30"
startsecs=1
autorestart=false
stopwaitsecs=5
`)
	// Needs ~1s to shut down after SIGTERM, well within its stopwaitsecs.
	slow := newTestProcess(t, "slow", `[program:slow]
command=/bin/sh -c "trap 'sleep 1; exit 0' TERM; while :; do sleep 0.1; done"
startsecs=1
autorestart=false
stopwaitsecs=5
`)
	hook := logtest.NewGlobal()
	defer hook.Reset()

	for _, p := range []*Process{fast, slow} {
		p.Start(false)
	}
	for _, p := range []*Process{fast, slow} {
		waitForState(t, p, Running, 10*time.Second)
	}

	var wg sync.WaitGroup
	for _, p := range []*Process{fast, slow} {
		wg.Add(1)
		go func(p *Process) {
			defer wg.Done()
			p.Stop(true)
		}(p)
	}
	wg.Wait()

	if ws := exitStatus(slow); ws.Signaled() || ws.ExitStatus() != 0 {
		t.Errorf("slow: want a clean exit 0 after its own TERM handler, got %v", ws)
	}
	if ws := exitStatus(fast); !ws.Signaled() || ws.Signal() != syscall.SIGTERM {
		t.Errorf("fast: want it ended by SIGTERM, got %v", ws)
	}
	for _, p := range []*Process{fast, slow} {
		if forceKilled(hook, p.GetName()) {
			t.Errorf("%s was force-killed although it exited within stopwaitsecs", p.GetName())
		}
	}
}

// A program that ignores its stop signal is still killed once stopwaitsecs
// is over.
func TestStopKillsProgramThatIgnoresStopSignal(t *testing.T) {
	p := newTestProcess(t, "stubborn", `[program:stubborn]
command=/bin/sh -c "trap '' TERM; while :; do sleep 0.1; done"
startsecs=1
autorestart=false
stopwaitsecs=1
`)
	hook := logtest.NewGlobal()
	defer hook.Reset()

	p.Start(false)
	waitForState(t, p, Running, 10*time.Second)

	start := time.Now()
	p.Stop(true)
	elapsed := time.Since(start)

	if ws := exitStatus(p); !ws.Signaled() || ws.Signal() != syscall.SIGKILL {
		t.Errorf("want SIGKILL after stopwaitsecs, got %v", ws)
	}
	if elapsed < 900*time.Millisecond {
		t.Errorf("killed after %v, before stopwaitsecs (1s)", elapsed)
	}
	if !forceKilled(hook, p.GetName()) {
		t.Error("expected a \"force to kill\" log entry")
	}
}
