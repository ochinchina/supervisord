package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/ochinchina/supervisord/config"
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

// StartGroupCommand start the given process group
type StartGroupCommand struct {
	Args struct {
		Groups []string `positional-arg-name:"Group" description:"Name of the Process Group"`
	} `positional-args:"yes" required:"yes"`
}

// StopGroupCommand stop the given process group
type StopGroupCommand struct {
	Args struct {
		Groups []string `positional-arg-name:"Group" description:"Name of the Process Group"`
	} `positional-args:"yes" required:"yes"`
}

// RestartCommand restart the given program
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

// PidCommand get the pid of program
type PidCommand struct {
	Args struct {
		Program string `positional-arg-name:"Program" description:"Name of the Program"`
	} `positional-args:"yes" required:"yes"`
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
	LogType string `short:"t" long:"type" choice:"stdout" choice:"stderr" description:"the log type, stdout or stderr" default:"stdout"`
	Args    struct {
		Program string `positional-arg-name:"Program" description:"Name of the Program"`
	} `positional-args:"yes" required:"yes"`
}

var ctlCommand CtlCommand
var statusCommand StatusCommand
var startCommand StartCommand
var stopCommand StopCommand
var startGroupCommand StartGroupCommand
var stopGroupCommand StopGroupCommand
var restartCommand RestartCommand
var shutdownCommand ShutdownCommand
var reloadCommand ReloadCommand
var pidCommand PidCommand
var signalCommand SignalCommand
var logtailCommand LogtailCommand

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
	if len(args) == 0 {
		return nil
	}

	rpcc := x.createRPCClient()
	verb := args[0]

	switch verb {

	////////////////////////////////////////////////////////////////////////////////
	// STATUS
	////////////////////////////////////////////////////////////////////////////////
	case "status":
		x.status(rpcc, args[1:])

		////////////////////////////////////////////////////////////////////////////////
		// START or STOP
		////////////////////////////////////////////////////////////////////////////////
	case "start", "stop":
		x.startStopProcesses(rpcc, verb, args[1:])
	case "start-group", "stop-group":
		x.startStopProcessGroups(rpcc, strings.Split(verb, "-")[0], args[1:])
	case "restart":
		x.restartProcesses(rpcc, args[1:])

		////////////////////////////////////////////////////////////////////////////////
		// SHUTDOWN
		////////////////////////////////////////////////////////////////////////////////
	case "shutdown":
		x.shutdown(rpcc)
	case "reload":
		x.reload(rpcc)
	case "signal":
		sigName, processes := args[1], args[2:]
		x.signal(rpcc, sigName, processes)
	case "pid":
		x.getPid(rpcc, args[1])
	default:
		fmt.Println("unknown command")
	}

	return nil
}

// get the status of processes
func (x *CtlCommand) status(rpcc *xmlrpcclient.XMLRPCClient, processes []string) {
	processesMap := make(map[string]bool)
	for _, process := range processes {
		processesMap[process] = true
	}
	if reply, err := rpcc.GetAllProcessInfo(); err == nil {
		x.showProcessInfo(&reply, processesMap)
	} else {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

// start or stop the processes
// verb must be: start or stop
func (x *CtlCommand) startStopProcesses(rpcc *xmlrpcclient.XMLRPCClient, verb string, processes []string) {
	state := map[string]string{
		"start": "started",
		"stop":  "stopped",
	}
	x._startStopProcesses(rpcc, verb, processes, state[verb], true)
}

func (x *CtlCommand) startStopProcessGroups(rpcc *xmlrpcclient.XMLRPCClient, verb string, groups []string) {
	state := map[string]string{
		"start": "started",
		"stop":  "stopped",
	}
	x._startStopProcessGroups(rpcc, verb, groups, state[verb], true)
}

func (x *CtlCommand) _startStopProcesses(rpcc *xmlrpcclient.XMLRPCClient, verb string, processes []string, state string, showProcessInfo bool) {
	if len(processes) <= 0 {
		fmt.Printf("Please specify process for %s\n", verb)
	}
	for _, pname := range processes {
		if pname == "all" {
			reply, err := rpcc.ChangeAllProcessState(verb)
			if err == nil {
				if showProcessInfo {
					x.showProcessInfo(&reply, make(map[string]bool))
				}
			} else {
				fmt.Printf("Fail to change all process state to %s", state)
			}
		} else {
			if reply, err := rpcc.ChangeProcessState(verb, pname); err == nil {
				if showProcessInfo {
					fmt.Printf("%s: ", pname)
					if !reply.Value {
						fmt.Printf("not ")
					}
					fmt.Printf("%s\n", state)
				}
			} else {
				fmt.Printf("%s: failed [%v]\n", pname, err)
				os.Exit(1)
			}
		}
	}
}

func (x *CtlCommand) _startStopProcessGroups(rpcc *xmlrpcclient.XMLRPCClient, verb string, groups []string, state string, showProcessInfo bool) {
	if len(groups) <= 0 {
		fmt.Printf("Please specify group for %s\n", verb)
	}
	for _, gname := range groups {
		if reply, err := rpcc.ChangeProcessGroupState(verb, gname); err == nil {
			if showProcessInfo {
				x.showProcessInfo(&reply, make(map[string]bool))
			}
		} else {
			fmt.Printf("%s: failed [%v]\n", gname, err)

		}
	}
}

func (x *CtlCommand) restartProcesses(rpcc *xmlrpcclient.XMLRPCClient, processes []string) {
	x._startStopProcesses(rpcc, "stop", processes, "stopped", false)
	x._startStopProcesses(rpcc, "start", processes, "restarted", true)
}

// shutdown the supervisord
func (x *CtlCommand) shutdown(rpcc *xmlrpcclient.XMLRPCClient) {
	if reply, err := rpcc.Shutdown(); err == nil {
		if reply.Value {
			fmt.Printf("Shut Down\n")
		} else {
			fmt.Printf("Hmmm! Something gone wrong?!\n")
		}
	} else {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

// reload all the programs in the supervisord
func (x *CtlCommand) reload(rpcc *xmlrpcclient.XMLRPCClient) {
	if reply, err := rpcc.ReloadConfig(); err == nil {

		if len(reply.AddedGroup) > 0 {
			fmt.Printf("Added Groups: %s\n", strings.Join(reply.AddedGroup, ","))
		}
		if len(reply.ChangedGroup) > 0 {
			fmt.Printf("Changed Groups: %s\n", strings.Join(reply.ChangedGroup, ","))
		}
		if len(reply.RemovedGroup) > 0 {
			fmt.Printf("Removed Groups: %s\n", strings.Join(reply.RemovedGroup, ","))
		}
	} else {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

// send signal to one or more processes
func (x *CtlCommand) signal(rpcc *xmlrpcclient.XMLRPCClient, sigName string, processes []string) {
	for _, process := range processes {
		if process == "all" {
			reply, err := rpcc.SignalAll(process)
			if err == nil {
				x.showProcessInfo(&reply, make(map[string]bool))
			} else {
				fmt.Printf("Fail to send signal %s to all process", sigName)
				os.Exit(1)
			}
		} else {
			reply, err := rpcc.SignalProcess(sigName, process)
			if err == nil && reply.Success {
				fmt.Printf("Succeed to send signal %s to process %s\n", sigName, process)
			} else {
				fmt.Printf("Fail to send signal %s to process %s\n", sigName, process)
				os.Exit(1)
			}
		}
	}
}

// get the pid of running program
func (x *CtlCommand) getPid(rpcc *xmlrpcclient.XMLRPCClient, process string) {
	procInfo, err := rpcc.GetProcessInfo(process)
	if err != nil {
		fmt.Printf("program '%s' not found\n", process)
		os.Exit(1)
	} else {
		fmt.Printf("%d\n", procInfo.Pid)
	}
}

func (x *CtlCommand) getProcessInfo(rpcc *xmlrpcclient.XMLRPCClient, process string) (types.ProcessInfo, error) {
	return rpcc.GetProcessInfo(process)
}

// check if group name should be displayed
func (x *CtlCommand) showGroupName() bool {
	val, ok := os.LookupEnv("SUPERVISOR_GROUP_DISPLAY")
	if !ok {
		return false
	}

	val = strings.ToLower(val)
	return val == "yes" || val == "true" || val == "y" || val == "t" || val == "1"
}

func (x *CtlCommand) showProcessInfo(reply *xmlrpcclient.AllProcessInfoReply, processesMap map[string]bool) {
	for _, pinfo := range reply.Value {
		description := pinfo.Description
		if strings.ToLower(description) == "<string></string>" {
			description = ""
		}
		if x.inProcessMap(&pinfo, processesMap) {
			processName := pinfo.GetFullName()
			if !x.showGroupName() {
				processName = pinfo.Name
			}
			fmt.Printf("%s%-33s%-10s%s%s\n", x.getANSIColor(strings.ToUpper(pinfo.Statename)), processName, pinfo.Statename, description, "\x1b[0m")
		}
	}
}

func (x *CtlCommand) inProcessMap(procInfo *types.ProcessInfo, processesMap map[string]bool) bool {
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

func (x *CtlCommand) logTail(rpcc *xmlrpcclient.XMLRPCClient, program string, logType string) {
	log, err := rpcc.TailProcessLog(program, 0, 10240, logType)
	if err != nil {
		fmt.Printf("Fail to tail log of program %s: %v\n", program, err)
		os.Exit(1)
	}
	os.Stdout.WriteString(log.LogData)

}

func (x *CtlCommand) getANSIColor(statename string) string {
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

// Execute implements flags.Commander interface to get status of program
func (sc *StatusCommand) Execute(args []string) error {
	ctlCommand.status(ctlCommand.createRPCClient(), sc.Args.Programs)
	return nil
}

// Execute start the given programs
func (sc *StartCommand) Execute(args []string) error {
	ctlCommand.startStopProcesses(ctlCommand.createRPCClient(), "start", sc.Args.Programs)
	return nil
}

// Execute stop the given programs
func (sc *StopCommand) Execute(args []string) error {
	ctlCommand.startStopProcesses(ctlCommand.createRPCClient(), "stop", sc.Args.Programs)
	return nil
}

// Execute start the given process group
func (sc *StartGroupCommand) Execute(args []string) error {
	ctlCommand.startStopProcessGroups(ctlCommand.createRPCClient(), "start", sc.Args.Groups)
	return nil
}

func (sc *StopGroupCommand) Execute(args []string) error {
	ctlCommand.startStopProcessGroups(ctlCommand.createRPCClient(), "stop", sc.Args.Groups)
	return nil
}

// Execute restart the programs
func (rc *RestartCommand) Execute(args []string) error {
	ctlCommand.restartProcesses(ctlCommand.createRPCClient(), rc.Args.Programs)
	return nil
}

// Execute shutdown the supervisor
func (sc *ShutdownCommand) Execute(args []string) error {
	ctlCommand.shutdown(ctlCommand.createRPCClient())
	return nil
}

// Execute stop the running programs and reload the supervisor configuration
func (rc *ReloadCommand) Execute(args []string) error {
	ctlCommand.reload(ctlCommand.createRPCClient())
	return nil
}

// Execute send signal to program
func (rc *SignalCommand) Execute(args []string) error {
	//sigName, processes := args[0], args[1:]
	ctlCommand.signal(ctlCommand.createRPCClient(), rc.Args.Signal, rc.Args.Programs)
	return nil
}

// Execute get the pid of program
func (pc *PidCommand) Execute(args []string) error {
	ctlCommand.getPid(ctlCommand.createRPCClient(), pc.Args.Program)
	return nil
}

// Execute tail the stdout/stderr of a program through http interface
func (lc *LogtailCommand) Execute(args []string) error {
	ctlCommand.logTail(ctlCommand.createRPCClient(), lc.Args.Program, lc.LogType)
	return nil
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
	_, _ = ctlCmd.AddCommand("start",
		"start programs",
		"start one or more programs",
		&startCommand)
	_, _ = ctlCmd.AddCommand("stop",
		"stop programs",
		"stop one or more programs",
		&stopCommand)
	_, _ = ctlCmd.AddCommand("start-group",
		"start a group of programs",
		"start one or more program groups",
		&startGroupCommand)
	_, _ = ctlCmd.AddCommand("stop-group",
		"stop a group of programs",
		"stop one or more program groups",
		&stopGroupCommand)
	_, _ = ctlCmd.AddCommand("restart",
		"restart programs",
		"restart one or more programs",
		&restartCommand)
	_, _ = ctlCmd.AddCommand("shutdown",
		"shutdown supervisord",
		"shutdown supervisord",
		&shutdownCommand)
	_, _ = ctlCmd.AddCommand("reload",
		"reload the programs",
		"reload the programs",
		&reloadCommand)
	_, _ = ctlCmd.AddCommand("signal",
		"send signal to program",
		"send signal to program",
		&signalCommand)
	_, _ = ctlCmd.AddCommand("pid",
		"get the pid of specified program",
		"get the pid of specified program",
		&pidCommand)
	_, _ = ctlCmd.AddCommand("logtail",
		"get the standard output&standard error of the program",
		"get the standard output&standard error of the program",
		&logtailCommand)
}
