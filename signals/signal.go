//go:build !windows && !darwin && !freebsd && !aix
// +build !windows,!darwin,!freebsd,!aix

package signals

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

var signalMap = map[string]os.Signal{"SIGABRT": syscall.SIGABRT,
	"SIGALRM":   syscall.SIGALRM,
	"SIGBUS":    syscall.SIGBUS,
	"SIGCHLD":   syscall.SIGCHLD,
	"SIGCLD":    syscall.SIGCLD,
	"SIGCONT":   syscall.SIGCONT,
	"SIGFPE":    syscall.SIGFPE,
	"SIGHUP":    syscall.SIGHUP,
	"SIGILL":    syscall.SIGILL,
	"SIGINT":    syscall.SIGINT,
	"SIGIO":     syscall.SIGIO,
	"SIGIOT":    syscall.SIGIOT,
	"SIGKILL":   syscall.SIGKILL,
	"SIGPIPE":   syscall.SIGPIPE,
	"SIGPOLL":   syscall.SIGPOLL,
	"SIGPROF":   syscall.SIGPROF,
	"SIGPWR":    syscall.SIGPWR,
	"SIGQUIT":   syscall.SIGQUIT,
	"SIGSEGV":   syscall.SIGSEGV,
	"SIGSTKFLT": syscall.SIGSTKFLT,
	"SIGSTOP":   syscall.SIGSTOP,
	"SIGSYS":    syscall.SIGSYS,
	"SIGTERM":   syscall.SIGTERM,
	"SIGTRAP":   syscall.SIGTRAP,
	"SIGTSTP":   syscall.SIGTSTP,
	"SIGTTIN":   syscall.SIGTTIN,
	"SIGTTOU":   syscall.SIGTTOU,
	"SIGUNUSED": syscall.SIGUNUSED,
	"SIGURG":    syscall.SIGURG,
	"SIGUSR1":   syscall.SIGUSR1,
	"SIGUSR2":   syscall.SIGUSR2,
	"SIGVTALRM": syscall.SIGVTALRM,
	"SIGWINCH":  syscall.SIGWINCH,
	"SIGXCPU":   syscall.SIGXCPU,
	"SIGXFSZ":   syscall.SIGXFSZ}

// ToSignal returns OS dependent signal name for given signal name (or syscall.SIGTERM if garbage given)
func ToSignal(signalName string) (os.Signal, error) {
	if !strings.HasPrefix(signalName, "SIG") {
		signalName = fmt.Sprintf("SIG%s", signalName)
	}
	if sig, ok := signalMap[signalName]; ok {
		return sig, nil
	}
	return syscall.SIGTERM, nil
}

// Kill sends signal to the process
//
// Args:
//
//	process - the process which the signal should be sent to
//	sigs - the signals will be sent
//	sigChildren - true if the signal needs to be sent to the children also
func Kill(process *os.Process, sigs []string, sigChildren bool, stopWaitSecs int) error {

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGCHLD)
	defer signal.Stop(signalChan)

	for _, sigStr := range sigs {
		sig, err := ToSignal(sigStr)
		if err != nil {
			continue
		}
		localSig := sig.(syscall.Signal)
		pid := process.Pid
		if sigChildren {
			pid = -pid
		}

		err = syscall.Kill(pid, localSig)
		if err != nil {
			continue
		}
		if waitForExit(process.Pid, signalChan, time.Duration(stopWaitSecs)*time.Second) {
			return nil
		}
	}
	return nil

}

// waitForExit waits until the process pid has exited (and been reaped), or the
// timeout passes. SIGCHLD is delivered when *any* child exits, so it's only a
// hint to check again; the process itself is polled with kill(pid, 0).
func waitForExit(pid int, sigchld <-chan os.Signal, timeout time.Duration) bool {
	deadline := time.After(timeout)
	tick := time.NewTicker(100 * time.Millisecond)
	defer tick.Stop()
	for {
		if syscall.Kill(pid, 0) == syscall.ESRCH {
			return true
		}
		select {
		case <-sigchld:
		case <-tick.C:
		case <-deadline:
			return false
		}
	}
}
