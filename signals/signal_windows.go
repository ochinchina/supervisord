// +build windows

package signals

import (
	"errors"
	"fmt"
	log "github.com/sirupsen/logrus"
	"os"
	"os/exec"
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
		log.Warn("signal USR1 is not supported in windows")
		return nil, errors.New("signal USR1 is not supported in windows")
	} else if signalName == "USR2" {
		log.Warn("signal USR2 is not supported in windows")
		return nil, errors.New("signal USR2 is not supported in windows")
	} else {
		return syscall.SIGTERM, nil

	}

}

func Kill(process *os.Process, sig os.Signal) error {
	//Signal command can't kill children processes, call  taskkill command to kill them
	cmd := exec.Command("taskkill", "/F", "/T", "/PID", fmt.Sprintf("%d", process.Pid))
	err := cmd.Start()
	if err == nil {
		return cmd.Wait()
	}
	//if fail to find taskkill, fallback to normal signal
	return process.Signal(sig)
}
