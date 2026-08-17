//go:build windows
// +build windows

package signals

import (
	"os"
	"syscall"
	"testing"
)

func TestToSignal_HUP(t *testing.T) {
	sig, err := ToSignal("HUP")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sig != syscall.SIGHUP {
		t.Errorf("expected SIGHUP, got %v", sig)
	}
}

func TestToSignal_INT(t *testing.T) {
	sig, err := ToSignal("INT")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sig != syscall.SIGINT {
		t.Errorf("expected SIGINT, got %v", sig)
	}
}

func TestToSignal_QUIT(t *testing.T) {
	sig, err := ToSignal("QUIT")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sig != syscall.SIGQUIT {
		t.Errorf("expected SIGQUIT, got %v", sig)
	}
}

func TestToSignal_KILL(t *testing.T) {
	sig, err := ToSignal("KILL")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sig != syscall.SIGKILL {
		t.Errorf("expected SIGKILL, got %v", sig)
	}
}

func TestToSignal_USR1_NotSupported(t *testing.T) {
	sig, err := ToSignal("USR1")
	if err == nil {
		t.Fatal("expected error for USR1 on windows")
	}
	if sig != nil {
		t.Errorf("expected nil signal, got %v", sig)
	}
	if err.Error() != "signal USR1 is not supported in windows" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestToSignal_USR2_NotSupported(t *testing.T) {
	sig, err := ToSignal("USR2")
	if err == nil {
		t.Fatal("expected error for USR2 on windows")
	}
	if sig != nil {
		t.Errorf("expected nil signal, got %v", sig)
	}
	if err.Error() != "signal USR2 is not supported in windows" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestToSignal_DefaultReturnsSIGTERM(t *testing.T) {
	sig, err := ToSignal("TERM")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sig != syscall.SIGTERM {
		t.Errorf("expected SIGTERM, got %v", sig)
	}
}

func TestToSignal_UnknownReturnsSIGTERM(t *testing.T) {
	sig, err := ToSignal("UNKNOWN")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sig != syscall.SIGTERM {
		t.Errorf("expected SIGTERM for unknown signal, got %v", sig)
	}
}

func TestToSignal_EmptyStringReturnsSIGTERM(t *testing.T) {
	sig, err := ToSignal("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sig != syscall.SIGTERM {
		t.Errorf("expected SIGTERM for empty string, got %v", sig)
	}
}

func TestKill_InvalidProcess(t *testing.T) {
	// Use an invalid PID that shouldn't match any real process
	proc, err := os.FindProcess(-1)
	if err != nil {
		t.Skipf("cannot create process handle: %v", err)
	}
	// Kill with TERM should not return an error (taskkill failure is swallowed via continue)
	err = Kill(proc, []string{"TERM"}, false, 1)
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
}

func TestKill_EmptySignals(t *testing.T) {
	proc, err := os.FindProcess(-1)
	if err != nil {
		t.Skipf("cannot create process handle: %v", err)
	}
	err = Kill(proc, []string{}, false, 1)
	if err != nil {
		t.Errorf("expected nil error with empty signals, got %v", err)
	}
}
