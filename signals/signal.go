// +build !windows

package signals

import (
	"os"
	"syscall"
)

//convert a signal name to signal
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

func Kill(process *os.Process, sig os.Signal) error {
	localSig := sig.(syscall.Signal)
	return syscall.Kill(-process.Pid, localSig)
}
