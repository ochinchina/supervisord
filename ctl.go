package main

import (
	"fmt"
	"strings"
)

type CtlCommand struct {
	Host string `short:"h" long:"host" description:"host on which supervisord server is running." default:"localhost"`
	Port int    `short:"p" long:"port" description:"port which supervisord server is listening." default:"9001"`
}

var ctlCommand CtlCommand

func (x *CtlCommand) Execute(args []string) error {
	if len(args) == 0 {
		return nil
	}

	rpcc := NewXmlRPCClient(x.Host, x.Port)

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
