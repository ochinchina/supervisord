//go:build !windows && !darwin && !freebsd && !aix
// +build !windows,!darwin,!freebsd,!aix

package signals

import (
	"bufio"
	"os"
	"os/exec"
	"syscall"
	"testing"
	"time"
)

// startChild runs script after `echo ready` and returns once the child has
// printed it, so any traps set before that are in place before a signal is
// sent. The child is reaped in the background, like supervisord does with
// cmd.Wait(); the returned channel is closed once it's reaped.
func startChild(t *testing.T, setup, script string) (*exec.Cmd, <-chan struct{}) {
	t.Helper()
	cmd := exec.Command("/bin/sh", "-c", setup+" echo ready; "+script)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	ready := make(chan bool, 1)
	go func() { ready <- bufio.NewScanner(stdout).Scan() }()
	select {
	case <-ready:
	case <-time.After(5 * time.Second):
		t.Fatal("child didn't get ready")
	}
	reaped := make(chan struct{})
	go func() {
		_ = cmd.Wait()
		close(reaped)
	}()
	t.Cleanup(func() {
		_ = cmd.Process.Signal(syscall.SIGKILL)
		<-reaped
	})
	return cmd, reaped
}

// SIGCHLD is delivered when *any* child exits. A sibling exiting while Kill
// waits for its own process must not end the wait: before the fix, Kill
// returned on the sibling's SIGCHLD and the caller escalated to SIGKILL long
// before stopwaitsecs (ochinchina/supervisord#430).
func TestKillWaitIsNotEndedBySiblingExit(t *testing.T) {
	// Ignores SIGTERM (an ignored signal stays ignored across exec).
	target, _ := startChild(t, `trap "" TERM;`, "exec sleep 30")
	// Exits on its own while Kill is waiting.
	startChild(t, "", "sleep 0.3")

	const wait = 2 * time.Second
	start := time.Now()
	_ = Kill(target.Process, []string{"TERM"}, false, int(wait/time.Second))
	elapsed := time.Since(start)

	if elapsed < wait-200*time.Millisecond {
		t.Fatalf("Kill returned after %v, before its %v wait, although its process was still running", elapsed, wait)
	}
	if err := syscall.Kill(target.Process.Pid, 0); err != nil {
		t.Fatalf("target should still be running after ignoring SIGTERM: %v", err)
	}
}

// When the process itself exits on the signal, Kill returns right away
// instead of sitting out the whole wait.
func TestKillReturnsAsSoonAsTheProcessExits(t *testing.T) {
	target, reaped := startChild(t, "", "exec sleep 30")

	start := time.Now()
	_ = Kill(target.Process, []string{"TERM"}, false, 10)
	elapsed := time.Since(start)

	if elapsed > 2*time.Second {
		t.Fatalf("Kill took %v; it should return once the process exited", elapsed)
	}
	select {
	case <-reaped:
	case <-time.After(2 * time.Second):
		t.Fatal("process was not reaped after Kill returned")
	}
}

// With several stop signals, the next one is only sent once the previous one
// has had its full wait; a sibling exiting must not skip ahead.
func TestKillSendsNextSignalOnlyAfterTheWait(t *testing.T) {
	// Ignores TERM, exits on INT.
	target, reaped := startChild(t, `trap "" TERM; trap "exit 0" INT;`, "while :; do sleep 0.1; done")
	startChild(t, "", "sleep 0.3")

	start := time.Now()
	_ = Kill(target.Process, []string{"TERM", "INT"}, false, 1)
	elapsed := time.Since(start)

	if elapsed < 800*time.Millisecond {
		t.Fatalf("INT was sent after %v, before TERM's 1s wait was over", elapsed)
	}
	select {
	case <-reaped:
	case <-time.After(3 * time.Second):
		t.Fatal("process didn't exit on the second stop signal")
	}
}

func TestWaitForExitTimesOutWhileProcessRuns(t *testing.T) {
	target, _ := startChild(t, "", "exec sleep 30")
	sigchld := make(chan os.Signal, 1)
	sigchld <- syscall.SIGCHLD // a stray SIGCHLD must not count as an exit

	start := time.Now()
	if waitForExit(target.Process.Pid, sigchld, 500*time.Millisecond) {
		t.Fatal("waitForExit reported an exit for a running process")
	}
	if elapsed := time.Since(start); elapsed < 400*time.Millisecond {
		t.Fatalf("waitForExit returned after %v, before its timeout", elapsed)
	}
}
