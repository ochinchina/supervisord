package main

import (
	"bufio"
	"fmt"
	"net/http"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/ochinchina/supervisord/config"
	"github.com/ochinchina/supervisord/faults"
	"github.com/ochinchina/supervisord/types"
	"github.com/ochinchina/supervisord/xmlrpcclient"
)

// CtlCommand the entry of ctl command
type CtlCommand struct {
	ServerURL string `short:"s" long:"serverurl" description:"URL on which supervisord server is listening"`
	User      string `short:"u" long:"user" description:"the user name"`
	Password  string `short:"P" long:"password" description:"the password"`
	Verbose   bool   `short:"v" long:"verbose" description:"Show verbose debug information"`
}

func createProcessGroup(processInfos []types.ProcessInfo) *config.ProcessGroup {
	processGroup := config.NewProcessGroup()
	for _, processInfo := range processInfos {
		processGroup.Add(processInfo.Group, processInfo.Name)
	}
	return processGroup
}

func getANSIColorByStateName(statename string) string {
	switch statename {
	case "RUNNING":
		// green
		return "\x1b[0;32m"
	case "BACKOFF", "FATAL":
		// red
		return "\x1b[0;31m"
	default:
		// yellow
		return "\x1b[1;33m"
	}
}

// check if group name should be displayed
func showGroupName() bool {
	val, ok := os.LookupEnv("SUPERVISOR_GROUP_DISPLAY")
	if !ok {
		return true
	}

	val = strings.ToLower(val)
	return val == "yes" || val == "true" || val == "y" || val == "t" || val == "1"
}

func showProcessInfo(reply *xmlrpcclient.AllProcessInfoReply, processesMap map[string]bool) {
	for _, pinfo := range reply.Value {
		description := pinfo.Description
		if strings.ToLower(description) == "<string></string>" {
			description = ""
		}
		if inProcessMap(&pinfo, processesMap) {
			processName := pinfo.GetFullName()
			if !showGroupName() {
				processName = pinfo.Name
			}
			fmt.Printf("%s%-33s%-10s%s%s\n", getANSIColorByStateName(strings.ToUpper(pinfo.Statename)), processName, pinfo.Statename, description, "\x1b[0m")
		}
	}
}

func inProcessMap(procInfo *types.ProcessInfo, processesMap map[string]bool) bool {
	if len(processesMap) <= 0 {
		return true
	}
	for procName := range processesMap {
		if procName == procInfo.Name || procName == procInfo.GetFullName() {
			return true
		}

		// check the wildcast '*'
		pos := strings.Index(procName, ":")
		if pos != -1 {
			groupName := procName[0:pos]
			programName := procName[pos+1:]
			if programName == "*" && groupName == procInfo.Group {
				return true
			}
		}
	}
	return false
}

func showProcessStatus(processStatuses []types.ProcessStatus) {
	for _, pstatus := range processStatuses {
		processName := pstatus.GetFullName()
		if !showGroupName() {
			processName = pstatus.Name
		}
		fmt.Printf("%s%-33s%-10s%s\n", faults.GetANSIColorByFaultCode(pstatus.Status), processName, faults.FaultCodeToString(pstatus.Status), "\x1b[0m")
	}
}

type AddCommand struct {
	Args struct {
		Programs []string `positional-arg-name:"Program" description:"Name of the Program/Group"`
	} `positional-args:"yes" required:"yes"`
}

type RemoveCommand struct {
	Args struct {
		Programs []string `positional-arg-name:"Program" description:"Name of the Program/Group"`
	} `positional-args:"yes" required:"yes"`
}

type ClearCommand struct {
	Args struct {
		Programs []string `positional-arg-name:"Program" description:"Name of the Program"`
	} `positional-args:"yes" required:"yes"`
}

// StatusCommand get the status of all supervisor managed programs
type StatusCommand struct {
	Args struct {
		Programs []string `positional-arg-name:"Program" description:"Name of the Program"`
	} `positional-args:"yes" required:"no"`
}

// StartCommand start the given program
type StartCommand struct {
	Args struct {
		Programs []string `positional-arg-name:"Program" description:"Name of the Program"`
	} `positional-args:"yes" required:"yes"`
}

// StopCommand stop the given program
type StopCommand struct {
	Args struct {
		Programs []string `positional-arg-name:"Program" description:"Name of the Program"`
	} `positional-args:"yes" required:"yes"`
}

// RestartCommand restart the given program
//
// restart <name>
//
//	Restart a process Note: restart does not reread config files. For that, see reread and update.
//
// restart <gname>:*
//
//	Restart all processes in a group Note: restart does not reread config files. For that, see reread and update.
//
// restart <name> <name>
//
//	Restart multiple processes or groups Note: restart does not reread config files. For that, see reread and update.
//
// restart all
//
//	Restart all processes Note: restart does not reread config files. For that, see reread and update.
type RestartCommand struct {
	Args struct {
		Programs []string `positional-arg-name:"Program" description:"Name of the Program"`
	} `positional-args:"yes" required:"yes"`
}

// ShutdownCommand shutdown the supervisor
type ShutdownCommand struct {
}

// ReloadCommand reload all the programs
type ReloadCommand struct {
}

type RereadCommand struct {
}

type UpdateCommand struct {
	Args struct {
		Groups []string `positional-arg-name:"Group" description:"Name of the Process Group"`
	} `positional-args:"yes" required:"no"`
}

// PidCommand get the pid of program
type PidCommand struct {
	Args struct {
		Program string `positional-arg-name:"Program" description:"Name of the Program"`
	} `positional-args:"yes"`
}

// SignalCommand send signal of program
type SignalCommand struct {
	Args struct {
		Signal   string   `positional-arg-name:"Signal" description:"Name of the Signal"`
		Programs []string `positional-arg-name:"Program" description:"Name of the Program"`
	} `positional-args:"yes" required:"yes"`
}

// LogtailCommand tail the stdout/stderr log of program through http interface
type LogtailCommand struct {
	Continuous bool `short:"f" description:"Continuous tail the log"`
	Args       struct {
		Program string `positional‑arg:"0" positional-arg-name:"Program" description:"Name of the Program" required:"yes"`
		LogType string `positional‑arg:"1" positional-arg-name:"LogType" choice:"stdout" choice:"stderr" description:"the log type, stdout or stderr" default:"stdout"`
	} `positional-args:"yes"`
}

type ForegroundCommand struct {
	Args struct {
		Program string `positional-arg-name:"Program" description:"Name of the Program"`
	} `positional-args:"yes" required:"yes"`
}

var ctlCommand CtlCommand
var addCommand AddCommand
var removeCommand RemoveCommand
var clearCommand ClearCommand
var statusCommand StatusCommand
var startCommand StartCommand
var stopCommand StopCommand
var restartCommand RestartCommand
var updateCommand UpdateCommand
var shutdownCommand ShutdownCommand
var reloadCommand ReloadCommand
var rereadCommand RereadCommand
var pidCommand PidCommand
var signalCommand SignalCommand
var logtailCommand LogtailCommand
var foregroundCommand ForegroundCommand

func (x *CtlCommand) getServerURL() string {
	options.Configuration, _ = findSupervisordConf()

	if x.ServerURL != "" {
		return x.ServerURL
	} else if _, err := os.Stat(options.Configuration); err == nil {
		myconfig := config.NewConfig(options.Configuration)
		_, _ = myconfig.Load()
		if entry, ok := myconfig.GetSupervisorctl(); ok {
			serverurl := entry.GetString("serverurl", "")
			if serverurl != "" {
				return serverurl
			}
		}
	}
	return "http://localhost:9001"
}

func (x *CtlCommand) getUser() string {
	options.Configuration, _ = findSupervisordConf()

	if x.User != "" {
		return x.User
	} else if _, err := os.Stat(options.Configuration); err == nil {
		myconfig := config.NewConfig(options.Configuration)
		_, _ = myconfig.Load()
		if entry, ok := myconfig.GetSupervisorctl(); ok {
			user := entry.GetString("username", "")
			return user
		}
	}
	return ""
}

func (x *CtlCommand) getPassword() string {
	options.Configuration, _ = findSupervisordConf()

	if x.Password != "" {
		return x.Password
	} else if _, err := os.Stat(options.Configuration); err == nil {
		myconfig := config.NewConfig(options.Configuration)
		_, _ = myconfig.Load()
		if entry, ok := myconfig.GetSupervisorctl(); ok {
			password := entry.GetString("password", "")
			return password
		}
	}
	return ""
}

func (x *CtlCommand) createRPCClient() *xmlrpcclient.XMLRPCClient {
	rpcc := xmlrpcclient.NewXMLRPCClient(x.getServerURL(), x.Verbose)
	rpcc.SetUser(x.getUser())
	rpcc.SetPassword(x.getPassword())
	return rpcc
}

// Execute implements flags.Commander interface to execute the control commands
func (x *CtlCommand) Execute(args []string) error {

	return nil
}

func (ac *AddCommand) Execute(args []string) error {
	client := ctlCommand.createRPCClient()
	failedPrograms := 0

	for _, program := range ac.Args.Programs {
		reply, err := client.AddProcessGroup(program)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Fail to add process group %s: %v\n", program, err)
			failedPrograms += 1
		} else if !reply.Success {
			fmt.Fprintf(os.Stderr, "Fail to add process group %s\n", program)
			failedPrograms += 1
		} else {
			fmt.Printf("Process group %s is added successfully\n", program)
		}
	}

	if failedPrograms > 0 {
		return fmt.Errorf("Fail to add %d groups", failedPrograms)
	}
	return nil
}

func (rc *RemoveCommand) Execute(args []string) error {

	client := ctlCommand.createRPCClient()
	failedPrograms := 0
	for _, program := range rc.Args.Programs {
		reply, err := client.RemoveProcessGroup(program)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Fail to remove process group %s: %v\n", program, err)
			failedPrograms += 1
		} else if !reply.Success {
			fmt.Fprintf(os.Stderr, "Fail to remove process group %s\n", program)
			failedPrograms += 1
		} else {
			fmt.Printf("Process group %s is removed successfully\n", program)
		}
	}
	if failedPrograms > 0 {
		os.Exit(1)
	}
	return nil
}

func (cc *ClearCommand) Execute(args []string) error {

	client := ctlCommand.createRPCClient()

	// clear all
	if slices.Contains(cc.Args.Programs, "all") {

		reply, err := client.ClearAllProcessLogs()
		if err != nil {
			fmt.Printf("Fail to clear all process logs: %v\n", err)
		} else {
			for _, procStatus := range reply.ProcessStatuses {
				if procStatus.Status == faults.Success {
					fmt.Printf("%s: cleared\n", procStatus.GetFullName())
				} else {
					fmt.Printf("%s: fail to clear\n", procStatus.GetFullName())
				}
			}
		}

	} else {
		for _, program := range cc.Args.Programs {

			reply, err := client.ClearProcessLogs(program)
			if err != nil {
				fmt.Printf("Fail to clear logs of program %s: %v\n", program, err)
			} else if reply.Success {
				fmt.Printf("Succeed to clear logs of program %s\n", program)
			} else {
				fmt.Printf("Fail to clear logs of program %s\n", program)
			}

		}
	}
	return nil
}

// Execute implements flags.Commander interface to get status of program
func (sc *StatusCommand) Execute(args []string) error {
	client := ctlCommand.createRPCClient()

	processesMap := make(map[string]bool)
	for _, process := range sc.Args.Programs {
		processesMap[process] = true
	}
	if reply, err := client.GetAllProcessInfo(); err == nil {
		showProcessInfo(&reply, processesMap)
	} else {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}

	return nil
}

// Execute start the given programs
func (sc *StartCommand) Execute(args []string) error {
	client := ctlCommand.createRPCClient()
	if len(sc.Args.Programs) == 0 || slices.Contains(sc.Args.Programs, "all") {
		reply, err := client.StartAllProcess(true)
		if err != nil {
			fmt.Printf("Fail to start all the processes with error:%v\n", err)
		} else {
			showProcessStatus(reply.ProcessStatuses)
		}
	} else {
		for _, program := range sc.Args.Programs {
			if strings.HasSuffix(program, ":*") {
				group := program[0 : len(program)-2]
				reply, err := client.StartProcessGroup(group, true)
				if err != nil {
					fmt.Printf("Fail to start group %s with error:%v\n", group, err)
				} else {
					fmt.Printf("Start status of group %s:\n", group)
					showProcessStatus(reply.ProcessStatuses)
				}
			} else {
				reply, err := client.StartProcess(program, true)
				if err != nil {
					fmt.Printf("Fail to start program %s with error:%v\n", program, err)
				} else if !reply.Success {
					fmt.Printf("Fail to start program %s\n", program)
				} else {
					fmt.Printf("Succeed to start program %s\n", program)
				}
			}
		}
	}

	return nil
}

// Execute stop the given programs
func (sc *StopCommand) Execute(args []string) error {
	client := ctlCommand.createRPCClient()

	if len(sc.Args.Programs) == 0 || slices.Contains(sc.Args.Programs, "all") {
		reply, err := client.StopAllProcess(true)
		if err != nil {
			fmt.Printf("Fail to stop all the processes with error:%v\n", err)
		} else {
			showProcessStatus(reply.ProcessStatuses)
		}
	} else {
		for _, program := range sc.Args.Programs {
			if strings.HasSuffix(program, ":*") {
				group := program[0 : len(program)-2]
				reply, err := client.StopProcessGroup(group, true)
				if err != nil {
					fmt.Printf("Fail to stop group %s with error:%v\n", group, err)
				} else {
					fmt.Printf("Stop status of group %s:\n", group)
					showProcessStatus(reply.ProcessStatuses)
				}
			} else {
				reply, err := client.StopProcess(program, true)
				if err != nil {
					fmt.Printf("Fail to stop program %s with error:%v\n", program, err)
				} else if !reply.Success {
					fmt.Printf("Fail to stop program %s\n", program)
				} else {
					fmt.Printf("Succeed to stop program %s\n", program)
				}
			}
		}
	}

	return nil
}

// Execute restart the programs
func (rc *RestartCommand) Execute(args []string) error {
	client := ctlCommand.createRPCClient()

	if len(rc.Args.Programs) == 0 || slices.Contains(rc.Args.Programs, "all") {
		rc.restartAllProcess(client)

	} else {
		for _, program := range rc.Args.Programs {
			rc.restartProcess(client, program)
		}
	}

	return nil
}

func (rc *RestartCommand) restartProcess(client *xmlrpcclient.XMLRPCClient, program string) {
	if strings.HasSuffix(program, ":*") {
		groupName := program[0 : len(program)-2]
		reply, err := client.StopProcessGroup(groupName, true)
		if err != nil {
			fmt.Printf("Fail to stop group %s with error:%v\n", groupName, err)
		} else {
			fmt.Printf("Stop status of group %s:\n", groupName)
			showProcessStatus(reply.ProcessStatuses)
		}

		time.Sleep(1 * time.Second)

		reply, err = client.StartProcessGroup(groupName, true)
		if err != nil {
			fmt.Printf("Fail to start group %s with error:%v\n", groupName, err)
		} else {
			fmt.Printf("Start status of group %s:\n", groupName)
			showProcessStatus(reply.ProcessStatuses)
		}
		return
	} else {
		reply, err := client.StopProcess(program, true)
		if err != nil {
			fmt.Printf("Fail to stop program %s with error:%v\n", program, err)
		} else if !reply.Success {
			fmt.Printf("Fail to stop program %s\n", program)
		} else {
			fmt.Printf("Succeed to stop program %s\n", program)
		}

		time.Sleep(1 * time.Second)

		reply, err = client.StartProcess(program, true)
		if err != nil {
			fmt.Printf("Fail to start program %s with error:%v\n", program, err)
		} else if !reply.Success {
			fmt.Printf("Fail to start program %s\n", program)
		} else {
			fmt.Printf("Succeed to start program %s\n", program)
		}
	}
}

func (rc *RestartCommand) restartAllProcess(client *xmlrpcclient.XMLRPCClient) {
	reply, err := client.StopAllProcess(true)
	if err != nil {
		fmt.Printf("Fail to stop all the processes with error:%v\n", err)
	} else {
		fmt.Printf("Stop status:\n")
		showProcessStatus(reply.ProcessStatuses)
	}

	time.Sleep(1 * time.Second)

	reply, err = client.StartAllProcess(true)
	if err != nil {
		fmt.Printf("Fail to start all the processes with error:%v\n", err)
	} else {
		fmt.Printf("Start status:\n")
		showProcessStatus(reply.ProcessStatuses)
	}
}

// Execute update the supervisord configuration and start the affected programs
func (rc *UpdateCommand) Execute(args []string) error {
	client := ctlCommand.createRPCClient()
	result, err := client.ReloadConfig()

	if err != nil {
		fmt.Fprintf(os.Stderr, "Fail to reload config: %s\n", err)
		os.Exit(1)
	}
	groups := rc.Args.Groups

	for _, group := range result.AddedGroup {
		if rc.ShouldOperateOnGroup(group, groups) {
			reply, err := client.AddProcessGroup(group)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Fail to add process group %s: %v\n", group, err)
			} else if !reply.Success {
				fmt.Fprintf(os.Stderr, "Fail to add process group %s\n", group)
			} else {
				fmt.Printf("Process group %s is added successfully\n", group)
			}
		}
	}

	if len(result.ChangedGroup) > 0 {
		allProcesses, _ := client.GetAllProcessInfo()
		for _, group := range result.ChangedGroup {
			if rc.ShouldOperateOnGroup(group, groups) {
				reply, err := client.AddProcessGroup(group)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Fail to add process group %s: %v\n", group, err)
				} else if !reply.Success {
					fmt.Fprintf(os.Stderr, "Fail to add process group %s\n", group)
				} else {
					fmt.Printf("Process group %s is added successfully\n", group)
					for _, procInfo := range allProcesses.Value {
						if procInfo.Group == group {
							_, _ = client.StopProcess(procInfo.Name, true)
							r, err := client.StartProcess(procInfo.Name, true)
							if err != nil {
								fmt.Fprintf(os.Stderr, "Fail to start process %s: %v\n", procInfo.Name, err)
							} else if !r.Success {
								fmt.Fprintf(os.Stderr, "Fail to start process %s\n", procInfo.Name)
							} else {
								fmt.Printf("Process %s is started successfully\n", procInfo.Name)
							}
						}
					}
				}
			}
		}
	}

	for _, group := range result.RemovedGroup {
		if rc.ShouldOperateOnGroup(group, groups) {
			_, err := client.StopProcessGroup(group, true)
			if err != nil {
				fmt.Printf("Fail to stop process group %s: %v\n", group, err)
			} else {
				fmt.Printf("Succeed to stop process group %s\n", group)
			}
			reply, err := client.RemoveProcessGroup(group)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Fail to remove process group %s: %v\n", group, err)
			} else if !reply.Success {
				fmt.Fprintf(os.Stderr, "Fail to remove process group %s\n", group)
			} else {
				fmt.Printf("Process group %s is removed successfully\n", group)
			}
		}
	}

	return nil
}

func (rc *UpdateCommand) ShouldOperateOnGroup(group string, groups []string) bool {
	if len(groups) == 0 || slices.Contains(groups, "all") {
		return true
	}
	return slices.Contains(groups, group)
}

// Execute shutdown the supervisor
func (sc *ShutdownCommand) Execute(args []string) error {

	client := ctlCommand.createRPCClient()

	if reply, err := client.Shutdown(); err == nil {
		if reply.Value {
			fmt.Printf("Succeed to shutdown\n")
		} else {
			fmt.Printf("Fail to shutdown\n")
		}
	} else {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}

	return nil
}

// Execute stop the running programs and reload the supervisor configuration
func (rc *ReloadCommand) Execute(args []string) error {

	client := ctlCommand.createRPCClient()

	if reply, err := client.Restart(); err == nil {

		if reply.Success {
			fmt.Printf("Supervisord is reloaded successfully\n")
		} else {
			fmt.Printf("Fail to reload supervisord\n")
		}
	} else {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
	return nil
}

func (rc *RereadCommand) Execute(args []string) error {

	client := ctlCommand.createRPCClient()
	if reply, err := client.ReloadConfig(); err == nil {
		if len(reply.AddedGroup) > 0 {
			fmt.Printf("Added Groups: %s\n", strings.Join(reply.AddedGroup, ","))
		} else {
			fmt.Printf("No new groups added\n")
		}
		if len(reply.ChangedGroup) > 0 {
			fmt.Printf("Changed Groups: %s\n", strings.Join(reply.ChangedGroup, ","))
		} else {
			fmt.Printf("No groups changed\n")
		}
		if len(reply.RemovedGroup) > 0 {
			fmt.Printf("Removed Groups: %s\n", strings.Join(reply.RemovedGroup, ","))
		} else {
			fmt.Printf("No groups removed\n")
		}
	} else {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		return fmt.Errorf("Fail to reread the configuration")
	}
	return nil
}

// Execute send signal to program
func (rc *SignalCommand) Execute(args []string) error {
	client := ctlCommand.createRPCClient()

	for _, process := range rc.Args.Programs {
		if process == "all" {
			reply, err := client.SignalAll(rc.Args.Signal)
			if err == nil {
				showProcessStatus(reply.ProcessStatuses)
			} else {
				fmt.Printf("Fail to send signal %s to all process", rc.Args.Signal)
				os.Exit(1)
			}
		} else if strings.HasSuffix(process, ":*") {
			group := process[:len(process)-2]
			reply, err := client.SignalProcessGroup(group, rc.Args.Signal)
			if err == nil {
				showProcessStatus(reply.ProcessStatuses)
			} else {
				fmt.Printf("Fail to send signal %s to group %s\n", rc.Args.Signal, group)
			}
		} else {
			reply, err := client.SignalProcess(rc.Args.Signal, process)
			if err == nil && reply.Success {
				fmt.Printf("Succeed to send signal %s to process %s\n", rc.Args.Signal, process)
			} else {
				fmt.Printf("Fail to send signal %s to process %s\n", rc.Args.Signal, process)
				os.Exit(1)
			}
		}
	}
	return nil
}

// Execute get the pid of program
func (pc *PidCommand) Execute(args []string) error {
	client := ctlCommand.createRPCClient()

	switch pc.Args.Program {
	case "":
		reply, err := client.GetSupervisordPID()
		if err != nil {
			fmt.Printf("Fail to get PID of supervisord with error:%v\n", err)
			os.Exit(1)
		} else {
			fmt.Printf("%d\n", reply.Pid)
		}
	case "all":
		reply, err := client.GetAllProcessInfo()
		if err != nil {
			fmt.Printf("Fail to get PID of all programs with error:%v\n", err)
			os.Exit(1)
		} else {
			for _, processInfo := range reply.Value {
				fmt.Printf("%s %d\n", processInfo.Name, processInfo.Pid)
			}
		}
	default:
		procInfo, err := client.GetProcessInfo(pc.Args.Program)
		if err != nil {
			fmt.Printf("program '%s' not found\n", pc.Args.Program)
			os.Exit(1)
		} else {
			fmt.Printf("%d\n", procInfo.Pid)
		}
	}
	return nil
}

// Execute tail the stdout/stderr of a program through http interface
func (lc *LogtailCommand) Execute(args []string) error {

	client := ctlCommand.createRPCClient()
	logType := "stdout"
	if lc.Args.LogType != "" {
		logType = lc.Args.LogType
	}

	if lc.Continuous {
		client := http.Client{Timeout: 30 * time.Second}
		req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/logtail/%s/%s", ctlCommand.getServerURL(), lc.Args.Program, logType), http.NoBody)
		if err != nil {
			fmt.Printf("Fail to create request to %s\n", ctlCommand.getServerURL())
			return err
		}
		if ctlCommand.getUser() != "" && ctlCommand.getPassword() != "" {
			req.SetBasicAuth(ctlCommand.getUser(), ctlCommand.getPassword())
		}
		resp, err := client.Do(req)
		if err != nil {
			fmt.Printf("Fail to connect to supervisord with error:%v\n", err)
			return err
		}
		defer resp.Body.Close()

		buffer := make([]byte, 1024)
		for {
			n, err := resp.Body.Read(buffer)
			if err != nil {
				return err
			}
			if lc.Args.LogType == "stdout" {
				os.Stdout.Write(buffer[:n])
			} else {
				os.Stderr.Write(buffer[:n])
			}
		}

	} else {
		if logType == "stdout" {
			log, err := client.ReadProcessStdoutLog(lc.Args.Program, -1600, 0)
			if err != nil {
				fmt.Printf("Fail to tail log of program %s: %v\n", lc.Args.Program, err)
				os.Exit(1)
			}
			os.Stdout.WriteString(log.LogData)
		} else {
			log, err := client.ReadProcessStderrLog(lc.Args.Program, -1600, 0)
			if err != nil {
				fmt.Printf("Fail to tail log of program %s: %v\n", lc.Args.Program, err)
				os.Exit(1)
			}
			os.Stdout.WriteString(log.LogData)
		}

	}

	return nil
}

func (fc *ForegroundCommand) Execute(args []string) error {

	client := ctlCommand.createRPCClient()

	procInfo, err := client.GetProcessInfo(fc.Args.Program)
	if err != nil {
		fmt.Printf("Fail to get process info of program %s: %v\n", fc.Args.Program, err)
		os.Exit(1)
	}
	if strings.ToUpper(procInfo.Statename) != "RUNNING" {
		fmt.Printf("Program '%s' is not running\n", fc.Args.Program)
		os.Exit(1)
	}

	os.Stdout.WriteString("\n\nEnter input to send to the program's stdin (Ctrl+C to exit):\n")

	// run the program in foreground
	go func() {
		logtailCommand := LogtailCommand{Continuous: true}
		logtailCommand.Args.Program = fc.Args.Program
		logtailCommand.Args.LogType = "stdout"
		logtailCommand.Execute(make([]string, 0))
	}()

	go func() {
		logtailCommand := LogtailCommand{Continuous: true}
		logtailCommand.Args.Program = fc.Args.Program
		logtailCommand.Args.LogType = "stderr"
		logtailCommand.Execute(make([]string, 0))
	}()

	for {
		reader := bufio.NewReader(os.Stdin)
		line, _, err := reader.ReadLine()
		if err != nil {
			os.Exit(1)
		}

		_, err = client.SendProcessStdin(fc.Args.Program, string(line)+"\n")
		if err != nil {
			fmt.Printf("Fail to send input to program %s: %v\n", fc.Args.Program, err)
			os.Exit(1)
		}
	}

}

func init() {
	ctlCmd, _ := parser.AddCommand("ctl",
		"Control a running daemon",
		"The ctl subcommand resembles supervisorctl command of original daemon.",
		&ctlCommand)
	_, _ = ctlCmd.AddCommand("status",
		"show program status",
		"show all or some program status",
		&statusCommand)
	_, _ = ctlCmd.AddCommand("add",
		"Activates any updates in config for process/group",
		"Activates any updates in config for process/group",
		&addCommand)
	_, _ = ctlCmd.AddCommand("remove",
		"Deactivates any updates in config for process/group",
		"Deactivates any updates in config for process/group",
		&removeCommand)
	_, _ = ctlCmd.AddCommand("clear",
		"clear the stdout/stderr log of the program",
		"clear the stdout/stderr log of the program",
		&clearCommand)
	_, _ = ctlCmd.AddCommand("start",
		"start programs",
		"start one or more programs",
		&startCommand)
	_, _ = ctlCmd.AddCommand("stop",
		"stop programs",
		"stop one or more programs",
		&stopCommand)
	_, _ = ctlCmd.AddCommand("restart",
		"restart programs",
		"restart one or more programs",
		&restartCommand)
	_, _ = ctlCmd.AddCommand("shutdown",
		"shutdown supervisord",
		"shutdown supervisord",
		&shutdownCommand)
	_, _ = ctlCmd.AddCommand("reload",
		"reload the supervisord configuration and start programs",
		"reload the supervisord configuration and start programs",
		&reloadCommand)
	_, _ = ctlCmd.AddCommand("reread",
		"Reload the daemon’s configuration files, without add/remove (no restarts)",
		"Reload the daemon’s configuration files, without add/remove (no restarts)",
		&rereadCommand)
	_, _ = ctlCmd.AddCommand("update",
		"reload the supervisord configuration and start programs",
		"reload the supervisord configuration and start programs",
		&updateCommand)
	_, _ = ctlCmd.AddCommand("signal",
		"send signal to program",
		"send signal to program",
		&signalCommand)
	_, _ = ctlCmd.AddCommand("pid",
		"get the pid of specified program",
		"get the pid of specified program",
		&pidCommand)
	_, _ = ctlCmd.AddCommand("tail",
		"get the standard output&standard error of the program",
		"get the standard output&standard error of the program",
		&logtailCommand)
	_, _ = ctlCmd.AddCommand("fg",
		"run the program in foreground",
		"run the program in foreground",
		&foregroundCommand)
}
