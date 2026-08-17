//go:build windows
// +build windows

package signals

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"
)

// convert a signal name to signal
func ToSignal(signalName string) (os.Signal, error) {
	switch signalName {
	case "HUP":
		return syscall.SIGHUP, nil
	case "INT":
		return syscall.SIGINT, nil
	case "QUIT":
		return syscall.SIGQUIT, nil
	case "KILL":
		return syscall.SIGKILL, nil
	case "USR1":
		log.Warn("signal USR1 is not supported in windows")
		return nil, errors.New("signal USR1 is not supported in windows")
	case "USR2":
		log.Warn("signal USR2 is not supported in windows")
		return nil, errors.New("signal USR2 is not supported in windows")
	default:
		return syscall.SIGTERM, nil

	}

}

// Args:
//
//	process - the process
//	sigs - the signals
//	sigChildren - ignore in windows system
func Kill(process *os.Process, sigs []string, sigChildren bool, stopWaitSecs int) error {
	//Signal command can't kill children processes, call  taskkill command to kill them
	done := make(chan error, 1)

	for index := range sigs {
		var cmd *exec.Cmd = nil
		if sigs[index] == "KILL" {
			cmd = exec.Command("taskkill", "/F", "/T", "/PID", fmt.Sprintf("%d", process.Pid))
		} else {
			cmd = exec.Command("taskkill", "/T", "/PID", fmt.Sprintf("%d", process.Pid))
		}
		go func() {
			output, err := cmd.CombinedOutput()
			if isKilledSucessfully(string(output)) {
				done <- nil
				return
			} else {
				done <- fmt.Errorf("taskkill failed: %v", err)
				return
			}
		}()
		// Wait for the process to exit or timeout
		select {
		// return if the process has exited
		case err := <-done:
			if err == nil {
				return nil
			} else {
				continue
			}
		// Timeout after stopWaitSecs seconds
		case <-time.After(time.Duration(stopWaitSecs) * time.Second):
			continue
		}
	}

	return fmt.Errorf("failed to kill process %d after %d seconds", process.Pid, stopWaitSecs)

}

func isKilledSucessfully(s string) bool {
	s = strings.TrimSpace(s)
	lines := strings.Split(strings.ReplaceAll(s, "\r\n", "\n"), "\n")

	for _, line := range lines {
		if strings.HasPrefix(line, "ERROR:") {
			log.Errorf("taskkill error: %s", line)
			return false
		}
	}
	return true
}
