package main

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"supervisord/config"
	"supervisord/types"
	"supervisord/xmlrpcclient"

	"github.com/spf13/cobra"
)

type CtlCommand struct {
	ServerURL string
	Username  string
	Password  string
	Verbose   bool
}

var (
	ctlOpt CtlCommand

	ctlCmd = cobra.Command{
		Use: "ctl",
	}
)

func init() {
	rootCmd.AddCommand(&ctlCmd)
	ctlCmd.PersistentFlags().StringVarP(&ctlOpt.ServerURL, "server-url", "s", "http://localhost:9001", "URL on which supervisord server is listening")
	ctlCmd.PersistentFlags().StringVarP(&ctlOpt.Username, "username", "u", "", "Username for authentication")
	ctlCmd.PersistentFlags().StringVarP(&ctlOpt.Password, "password", "p", "", "Password for authentication")
	ctlCmd.PersistentFlags().BoolVarP(&ctlOpt.Verbose, "verbose", "v", false, "Show verbose debug information")

	ctlCmd.AddCommand(&cobra.Command{
		Use: "status",
		Run: func(cmd *cobra.Command, args []string) {
			ctlOpt.status(args)
		},
	})

	ctlCmd.AddCommand(&cobra.Command{
		Use: "start",
		Run: func(cmd *cobra.Command, args []string) {
			ctlOpt.startStopProcesses("start", args)
		},
	})

	ctlCmd.AddCommand(&cobra.Command{
		Use: "stop",
		Run: func(cmd *cobra.Command, args []string) {
			ctlOpt.startStopProcesses("stop", args)
		},
	})

	ctlCmd.AddCommand(&cobra.Command{
		Use: "shutdown",
		Run: func(cmd *cobra.Command, args []string) {
			ctlOpt.shutdown()
		},
	})

	ctlCmd.AddCommand(&cobra.Command{
		Use: "reload",
		Run: func(cmd *cobra.Command, args []string) {
			ctlOpt.reload()
		},
	})

	ctlCmd.AddCommand(&cobra.Command{
		Use: "signal",
		Run: func(cmd *cobra.Command, args []string) {
			sigName, processes := args[0], args[1:]
			ctlOpt.signal(sigName, processes)
		},
		Args: cobra.MinimumNArgs(2),
	})

	ctlCmd.AddCommand(&cobra.Command{
		Use: "pid",
		Run: func(cmd *cobra.Command, args []string) {
			ctlOpt.getPid(args[0])
		},
		Args: cobra.ExactArgs(1),
	})

	ctlCmd.AddCommand(&cobra.Command{
		Use: "logtail",
		RunE: func(cmd *cobra.Command, args []string) error {
			program := args[0]
			go func() {
				ctlOpt.tailLog(program, "stderr")
			}()
			err := ctlOpt.tailLog(program, "stdout")
			if !errors.Is(err, io.EOF) {
				return err
			}
			return nil
		},
		Args: cobra.ExactArgs(1),
	})
}

func (x *CtlCommand) createRPCClient() *xmlrpcclient.XMLRPCClient {
	rpcc := xmlrpcclient.NewXMLRPCClient(x.getServerURL(), x.Verbose)
	rpcc.SetUser(x.getUser())
	rpcc.SetPassword(x.getPassword())
	return rpcc
}

func (x *CtlCommand) getServerURL() string {
	if x.ServerURL != "" {
		return x.ServerURL
	} else if _, err := os.Stat(rootOpt.Configuration); err == nil {
		config := config.NewConfig(rootOpt.Configuration)
		config.Load()
		if config.SupervisorCtl != nil && config.SupervisorCtl.ServerURL != "" {
			return config.SupervisorCtl.ServerURL
		}
	}
	return "http://localhost:9001"
}

func (x *CtlCommand) getUser() string {
	if x.Username != "" {
		return x.Username
	} else if _, err := os.Stat(rootOpt.Configuration); err == nil {
		config := config.NewConfig(rootOpt.Configuration)
		config.Load()
		if config.SupervisorCtl != nil {
			return config.SupervisorCtl.Username
		}
	}
	return ""
}

func (x *CtlCommand) getPassword() string {
	if x.Password != "" {
		return x.Password
	} else if _, err := os.Stat(rootOpt.Configuration); err == nil {
		config := config.NewConfig(rootOpt.Configuration)
		config.Load()
		if config.SupervisorCtl != nil {
			return config.SupervisorCtl.Password
		}
	}
	return ""
}

// get the status of processes
func (x *CtlCommand) status(processes []string) {
	rpcc := x.createRPCClient()
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
func (x *CtlCommand) startStopProcesses(verb string, processes []string) {
	rpcc := x.createRPCClient()
	state := map[string]string{
		"start": "started",
		"stop":  "stopped",
	}
	x._startStopProcesses(rpcc, verb, processes, state[verb], true)
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

func (x *CtlCommand) restartProcesses(processes []string) {
	rpcc := x.createRPCClient()
	x._startStopProcesses(rpcc, "stop", processes, "stopped", false)
	x._startStopProcesses(rpcc, "start", processes, "restarted", true)
}

// shutdown the supervisord
func (x *CtlCommand) shutdown() {
	rpcc := x.createRPCClient()
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
func (x *CtlCommand) reload() {
	rpcc := x.createRPCClient()
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
func (x *CtlCommand) signal(sigName string, processes []string) {
	rpcc := x.createRPCClient()
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
func (x *CtlCommand) getPid(process string) {
	rpcc := x.createRPCClient()
	procInfo, err := rpcc.GetProcessInfo(process)
	if err != nil {
		fmt.Printf("program '%s' not found\n", process)
		os.Exit(1)
	} else {
		fmt.Printf("%d\n", procInfo.Pid)
	}
}

func (x *CtlCommand) getProcessInfo(process string) (types.ProcessInfo, error) {
	rpcc := x.createRPCClient()
	return rpcc.GetProcessInfo(process)
}

func (x *CtlCommand) tailLog(program, dev string) error {
	_, err := x.getProcessInfo(program)
	if err != nil {
		fmt.Printf("Not exist program %s\n", program)
		return err
	}
	url := fmt.Sprintf("%s/logtail/%s/%s", x.getServerURL(), program, dev)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	req.SetBasicAuth(x.getUser(), x.getPassword())
	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	buf := make([]byte, 10240)
	for {
		n, err := resp.Body.Read(buf)
		if err != nil {
			return err
		}
		if dev == "stdout" {
			os.Stdout.Write(buf[0:n])
		} else {
			os.Stderr.Write(buf[0:n])
		}
	}
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
			fmt.Printf("%s%-33s%-10s%s%s\n", x.getANSIColor(pinfo.Statename), processName, pinfo.Statename, description, "\x1b[0m")
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
