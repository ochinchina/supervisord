// +build !windows

package signals

import (
	"os"
	"syscall"
)

// ToSignal convert a signal name to signal
func ToSignal(signalName string) (os.Signal, error) {
	if signalName == "HUP" {
		return syscall.SIGHUP, nil
	} else if signalName == "INT" {
		return syscall.SIGINT, nil
	} else if signalName == "QUIT" {
		return syscall.SIGQUIT, nil
	} else if signalName == "KILL" {
		return syscall.SIGKILL, nil
	} else if signalName == "USR1" {
		return syscall.SIGUSR1, nil
	} else if signalName == "USR2" {
		return syscall.SIGUSR2, nil
	} else {
		return syscall.SIGTERM, nil

	}

}

// Kill send signal to the process
//
// Args:
//    process - the process which the signal should be sent to
//    sig - the signal will be sent
//    sigChildren - true if the signal needs to be sent to the children also
//
func Kill(process *os.Process, sig os.Signal, sigChildren bool) error {
	localSig := sig.(syscall.Signal)
	pid := process.Pid
	if sigChildren {
		pid = -pid
	}
	return syscall.Kill(pid, localSig)
}
