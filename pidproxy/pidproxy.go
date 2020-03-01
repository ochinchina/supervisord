package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"time"
)

func installSignalAndForward(pidfile string, exitIfDaemonStopped bool) {
	c := make(chan os.Signal, 1)
	installSignal(c)

	timer := time.After(5 * time.Second)
	for {
		select {
		case sig := <-c:
			fmt.Printf("Get a signal %v\n", sig)
			if allowForwardSig(sig) {
				forwardSignal(sig, pidfile)
			}

			if sig == syscall.SIGTERM ||
				sig == syscall.SIGINT ||
				sig == syscall.SIGQUIT {
				os.Exit(0)
			}
		case <-timer:
			timer = time.After(5 * time.Second)
			pid, err := readPid(pidfile)
			if err == nil && !isProcessAlive(pid) {
				fmt.Printf("Process %d is not alive\n", pid)
				if exitIfDaemonStopped {
					os.Exit(1)
				}
			}

		}
	}
}

func isProcessAlive(pid int) bool {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	return proc.Signal(syscall.Signal(0)) == nil
}

func forwardSignal(sig os.Signal, pidfile string) {
	pid, err := readPid(pidfile)

	if err == nil {
		fmt.Printf("Read pid %d from file %s\n", pid, pidfile)
		proc, err := os.FindProcess(pid)
		if err == nil {
			err = proc.Signal(sig)
			if err == nil {
				fmt.Printf("Succeed to send signal %v to process %d\n", sig, pid)
				return
			}
		}
		fmt.Printf("Fail to send signal %v to process %d with error:%v\n", sig, pid, err)
	} else {
		fmt.Printf("Fail to read pid from file %s with error:%v\n", pidfile, err)
	}
}

func readPid(pidfile string) (int, error) {
	file, err := os.Open(pidfile)
	if err == nil {
		defer file.Close()
		pid := 0
		n, err := fmt.Fscanf(file, "%d", &pid)
		if err == nil {
			if n != 1 {
				return pid, errors.New("Fail to get pid from file")
			}
			return pid, nil
		}
	}
	return 0, err
}

func startApplication(command string, args []string) {
	cmd := exec.Command(command)
	for _, arg := range args {
		cmd.Args = append(cmd.Args, arg)
	}

	err := cmd.Start()

	if err == nil {
		err = cmd.Wait()
		if err == nil {
			fmt.Printf("Succeed to start program:%s\n", command)
			return
		}

	}
	fmt.Printf("Fail to start program with error %v\n", err)
	os.Exit(1)
}

func printUsage() {
	fmt.Println("Usage: pidproxy [-exit-daemon-stop] <pidfile> <command> [args...]")
	fmt.Println("exit-daemon-stop  exit this pidproxy if the started daemon exits")
}
func main() {
	var args []string
	exitIfDaemonStopped := false
	if os.Args[1] == "-exit-daemon-stop" {
		exitIfDaemonStopped = true
		args = os.Args[2:]
	} else {
		args = os.Args[1:]
	}

	if len(args) < 2 {
		printUsage()
	} else {
		pidfile := args[0]
		command := args[1]

		startApplication(command, args[2:])
		installSignalAndForward(pidfile, exitIfDaemonStopped)
	}
}
