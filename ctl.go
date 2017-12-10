package main

import (
	"fmt"
	"github.com/ochinchina/supervisord/xmlrpcclient"
	"strings"
)

type CtlCommand struct {
	ServerUrl string `short:"s" long:"serverurl" description:"URL on which supervisord server is listening" default:"http://localhost:9001"`
}

var ctlCommand CtlCommand

func (x *CtlCommand) Execute(args []string) error {
	if len(args) == 0 {
		return nil
	}

	rpcc := xmlclient.NewXmlRPCClient(x.ServerUrl)

	verb, processes := args[0], args[1:]
	hasProcesses := len(processes) > 0
	processesMap := make(map[string]bool)
	for _, process := range processes {
		processesMap[strings.ToLower(process)] = true
	}

	switch verb {

	////////////////////////////////////////////////////////////////////////////////
	// STATUS
	////////////////////////////////////////////////////////////////////////////////
	case "status":
		if reply, err := rpcc.GetAllProcessInfo(); err == nil {
			for _, pinfo := range reply.Value {
				name := strings.ToLower(pinfo.Name)
				description := pinfo.Description
				if strings.ToLower(description) == "<string></string>" {
					description = ""
				}
				if !hasProcesses || processesMap[name] {
					fmt.Printf("%-33s%-10s%s\n", name, pinfo.Statename, description)
				}
			}
		}

	////////////////////////////////////////////////////////////////////////////////
	// START or STOP
	////////////////////////////////////////////////////////////////////////////////
	case "start", "stop":
		state := map[string]string{
			"start": "started",
			"stop":  "stopped",
		}
		if len(processes) <= 0 {
			fmt.Printf("Please specify process for %s\n", verb)
		}
		for _, pname := range processes {
			if reply, err := rpcc.ChangeProcessState(verb, pname); err == nil {
				fmt.Printf("%s: ", pname)
				if !reply.Value {
					fmt.Printf("not ")
				}
				fmt.Printf("%s\n", state[verb])
			} else {
				fmt.Printf("%s: failed [%v]\n", pname, err)
			}
		}

	////////////////////////////////////////////////////////////////////////////////
	// SHUTDOWN
	////////////////////////////////////////////////////////////////////////////////
	case "shutdown":
		if reply, err := rpcc.Shutdown(); err == nil {
			if reply.Value {
				fmt.Printf("Shut Down\n")
			} else {
				fmt.Printf("Hmmm! Something gone wrong?!\n")
			}
		}
	case "reload":
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
		}
	default:
		fmt.Println("unknown command")
	}

	return nil
}

func init() {
	parser.AddCommand("ctl",
		"Control a running daemon",
		"The ctl subcommand resembles supervisorctl command of original daemon.",
		&ctlCommand)
}
