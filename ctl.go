package main

import (
	"fmt"
	"github.com/ochinchina/supervisord/config"
	"github.com/ochinchina/supervisord/types"
	"github.com/ochinchina/supervisord/xmlrpcclient"
	"os"
	"strings"
)

type CtlCommand struct {
	ServerUrl string `short:"s" long:"serverurl" description:"URL on which supervisord server is listening"`
	User      string `short:"u" long:"user" description:"the user name"`
	Password  string `short:"P" long:"password" description:"the password"`
	Verbose   bool   `short:"v" long:"verbose" description:"Show verbose debug information"`
}

type StatusCommand struct {
}

type StartCommand struct {
}

type StopCommand struct {
}

type ShutdownCommand struct {
}

type ReloadCommand struct {
}

type PidCommand struct {
}

type SignalCommand struct {
}

var ctlCommand CtlCommand
var statusCommand StatusCommand
var startCommand StartCommand
var stopCommand StopCommand
var shutdownCommand ShutdownCommand
var reloadCommand ReloadCommand
var pidCommand PidCommand
var signalCommand SignalCommand

func (x *CtlCommand) getServerUrl() string {
	options.Configuration, _ = findSupervisordConf()

	if x.ServerUrl != "" {
		return x.ServerUrl
	} else if _, err := os.Stat(options.Configuration); err == nil {
		config := config.NewConfig(options.Configuration)
		config.Load()
		if entry, ok := config.GetSupervisorctl(); ok {
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
		config := config.NewConfig(options.Configuration)
		config.Load()
		if entry, ok := config.GetSupervisorctl(); ok {
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
		config := config.NewConfig(options.Configuration)
		config.Load()
		if entry, ok := config.GetSupervisorctl(); ok {
			password := entry.GetString("password", "")
			return password
		}
	}
	return ""
}

func (x *CtlCommand) createRpcClient() *xmlrpcclient.XmlRPCClient {
	rpcc := xmlrpcclient.NewXmlRPCClient(x.getServerUrl(), x.Verbose)
	rpcc.SetUser(x.getUser())
	rpcc.SetPassword(x.getPassword())
	return rpcc
}

func (x *CtlCommand) Execute(args []string) error {
	if len(args) == 0 {
		return nil
	}

	rpcc := x.createRpcClient()
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

	////////////////////////////////////////////////////////////////////////////////
	// SHUTDOWN
	////////////////////////////////////////////////////////////////////////////////
	case "shutdown":
		x.shutdown(rpcc)
	case "reload":
		x.reload(rpcc)
	case "signal":
		sig_name, processes := args[1], args[2:]
		x.signal(rpcc, sig_name, processes)
	case "pid":
		x.getPid(rpcc, args[1])
	default:
		fmt.Println("unknown command")
	}

	return nil
}

// get the status of processes
func (x *CtlCommand) status(rpcc *xmlrpcclient.XmlRPCClient, processes []string) {
	processesMap := make(map[string]bool)
	for _, process := range processes {
		processesMap[process] = true
	}
	if reply, err := rpcc.GetAllProcessInfo(); err == nil {
		x.showProcessInfo(&reply, processesMap)
	} else {
		os.Exit(1)
	}
}

// start or stop the processes
// verb must be: start or stop
func (x *CtlCommand) startStopProcesses(rpcc *xmlrpcclient.XmlRPCClient, verb string, processes []string) {
	state := map[string]string{
		"start": "started",
		"stop":  "stopped",
	}
	if len(processes) <= 0 {
		fmt.Printf("Please specify process for %s\n", verb)
	}
	for _, pname := range processes {
		if pname == "all" {
			reply, err := rpcc.ChangeAllProcessState(verb)
			if err == nil {
				x.showProcessInfo(&reply, make(map[string]bool))
			} else {
				fmt.Printf("Fail to change all process state to %s", state)
			}
		} else {
			if reply, err := rpcc.ChangeProcessState(verb, pname); err == nil {
				fmt.Printf("%s: ", pname)
				if !reply.Value {
					fmt.Printf("not ")
				}
				fmt.Printf("%s\n", state[verb])
			} else {
				fmt.Printf("%s: failed [%v]\n", pname, err)
				os.Exit(1)
			}
		}
	}
}

// shutdown the supervisord
func (x *CtlCommand) shutdown(rpcc *xmlrpcclient.XmlRPCClient) {
	if reply, err := rpcc.Shutdown(); err == nil {
		if reply.Value {
			fmt.Printf("Shut Down\n")
		} else {
			fmt.Printf("Hmmm! Something gone wrong?!\n")
		}
	} else {
		os.Exit(1)
	}
}

// reload all the programs in the supervisord
func (x *CtlCommand) reload(rpcc *xmlrpcclient.XmlRPCClient) {
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
		os.Exit(1)
	}
}

// send signal to one or more processes
func (x *CtlCommand) signal(rpcc *xmlrpcclient.XmlRPCClient, sig_name string, processes []string) {
	for _, process := range processes {
		if process == "all" {
			reply, err := rpcc.SignalAll(process)
			if err == nil {
				x.showProcessInfo(&reply, make(map[string]bool))
			} else {
				fmt.Printf("Fail to send signal %s to all process", sig_name)
				os.Exit(1)
			}
		} else {
			reply, err := rpcc.SignalProcess(sig_name, process)
			if err == nil && reply.Success {
				fmt.Printf("Succeed to send signal %s to process %s\n", sig_name, process)
			} else {
				fmt.Printf("Fail to send signal %s to process %s\n", sig_name, process)
				os.Exit(1)
			}
		}
	}
}

// get the pid of running program
func (x *CtlCommand) getPid(rpcc *xmlrpcclient.XmlRPCClient, process string) {
	procInfo, err := rpcc.GetProcessInfo(process)
	if err != nil {
		fmt.Printf("program '%s' not found\n", process)
		os.Exit(1)
	} else {
		fmt.Printf("%d\n", procInfo.Pid)
	}
}

func (x *CtlCommand) showProcessInfo(reply *xmlrpcclient.AllProcessInfoReply, processesMap map[string]bool) {
	for _, pinfo := range reply.Value {
		description := pinfo.Description
		if strings.ToLower(description) == "<string></string>" {
			description = ""
		}
		if x.inProcessMap(&pinfo, processesMap) {
			fmt.Printf("%s%-33s%-10s%s%s\n", x.getANSIColor(pinfo.Statename), pinfo.GetFullName(), pinfo.Statename, description, "\x1b[0m")
		}
	}
}

func (x *CtlCommand) inProcessMap(procInfo *types.ProcessInfo, processesMap map[string]bool) bool {
	if len(processesMap) <= 0 {
		return true
	}
	for procName, _ := range processesMap {
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

func (x *CtlCommand) getANSIColor(statename string) string {
	if statename == "RUNNING" {
		// green
		return "\x1b[0;32m"
	} else if statename == "BACKOFF" || statename == "FATAL" {
		// red
		return "\x1b[0;31m"
	} else {
		// yellow
		return "\x1b[1;33m"
	}
}

func (sc *StatusCommand) Execute(args []string) error {
	ctlCommand.status(ctlCommand.createRpcClient(), args)
	return nil
}

func (sc *StartCommand) Execute(args []string) error {
	ctlCommand.startStopProcesses(ctlCommand.createRpcClient(), "start", args)
	return nil
}

func (sc *StopCommand) Execute(args []string) error {
	ctlCommand.startStopProcesses(ctlCommand.createRpcClient(), "stop", args)
	return nil
}

func (sc *ShutdownCommand) Execute(args []string) error {
	ctlCommand.shutdown(ctlCommand.createRpcClient())
	return nil
}

func (rc *ReloadCommand) Execute(args []string) error {
	ctlCommand.reload(ctlCommand.createRpcClient())
	return nil
}

func (rc *SignalCommand) Execute(args []string) error {
	sig_name, processes := args[0], args[1:]
	ctlCommand.signal(ctlCommand.createRpcClient(), sig_name, processes)
	return nil
}

func (pc *PidCommand) Execute(args []string) error {
	ctlCommand.getPid(ctlCommand.createRpcClient(), args[0])
	return nil
}

func init() {
	ctlCmd, _ := parser.AddCommand("ctl",
		"Control a running daemon",
		"The ctl subcommand resembles supervisorctl command of original daemon.",
		&ctlCommand)
	ctlCmd.AddCommand("status",
		"show program status",
		"show all or some program status",
		&statusCommand)
	ctlCmd.AddCommand("start",
		"start programs",
		"start one or more programs",
		&startCommand)
	ctlCmd.AddCommand("stop",
		"stop programs",
		"stop one or more programs",
		&stopCommand)
	ctlCmd.AddCommand("shutdown",
		"shutdown supervisord",
		"shutdown supervisord",
		&shutdownCommand)
	ctlCmd.AddCommand("reload",
		"reload the programs",
		"reload the programs",
		&reloadCommand)
	ctlCmd.AddCommand("signal",
		"send signal to program",
		"send signal to program",
		&signalCommand)
	ctlCmd.AddCommand("pid",
		"get the pid of specified program",
		"get the pid of specified program",
		&pidCommand)

}
