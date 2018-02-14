package main

import (
	"fmt"
	"github.com/ochinchina/supervisord/xmlrpcclient"
    "github.com/ochinchina/supervisord/config"
    "os"
	"strings"
)

type CtlCommand struct {
    ServerUrl string `short:"s" long:"serverurl" description:"URL on which supervisord server is listening"`
}

var ctlCommand CtlCommand

func (x *CtlCommand)getServerUrl() string{
    fmt.Printf("%v\n", options )
    if x.ServerUrl != "" {
        return x.ServerUrl
    } else if _, err := os.Stat( options.Configuration ); err == nil {
        config := config.NewConfig( options.Configuration )
        config.Load()
        if entry, ok := config.GetSupervisorctl(); ok {
            serverurl := entry.GetString( "serverurl", "" )
            if serverurl != "" {
                return serverurl
            }
        }
    }
    return "http://localhost:9001"
}
func (x *CtlCommand) Execute(args []string) error {
	if len(args) == 0 {
		return nil
	}

	rpcc := xmlrpcclient.NewXmlRPCClient(x.getServerUrl() )
    verb := args[0]

	switch verb {

	////////////////////////////////////////////////////////////////////////////////
	// STATUS
	////////////////////////////////////////////////////////////////////////////////
	case "status":
        processes := args[1:]
        processesMap := make(map[string]bool)
        for _, process := range processes {
            processesMap[strings.ToLower(process)] = true
        }
		if reply, err := rpcc.GetAllProcessInfo(); err == nil {
            x.showProcessInfo( &reply, processesMap )
		}

	////////////////////////////////////////////////////////////////////////////////
	// START or STOP
	////////////////////////////////////////////////////////////////////////////////
	case "start", "stop":
		state := map[string]string{
			"start": "started",
			"stop":  "stopped",
		}
        processes := args[1:]
		if len(processes) <= 0 {
			fmt.Printf("Please specify process for %s\n", verb)
		}
		for _, pname := range processes {
            if pname == "all" {
                reply, err := rpcc.ChangeAllProcessState( verb )
                if err == nil {
                    x.showProcessInfo( &reply, make(map[string]bool) )
                }else {
                    fmt.Printf( "Fail to change all process state to %s", state )
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
			    }
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
    case "signal":
        sig_name, processes := args[1], args[2:]
        for _, process := range( processes ) {
            if process == "all" {
                reply, err := rpcc.SignalAll( process )
                if err == nil {
                    x.showProcessInfo( &reply, make(map[string]bool) )
                } else {
                    fmt.Printf( "Fail to send signal %s to all process" , sig_name )
                }
            } else {
                reply, err := rpcc.SignalProcess( sig_name, process )
                if err == nil && reply.Success {
                   fmt.Printf( "Succeed to send signal %s to process %s\n", sig_name, process )
                } else {
                    fmt.Printf( "Fail to send signal %s to process %s\n", sig_name, process )
                }
            }
        }

	default:
		fmt.Println("unknown command")
	}

	return nil
}

func (x *CtlCommand)showProcessInfo( reply* xmlrpcclient.AllProcessInfoReply, processesMap map[string]bool ) {
    for _, pinfo := range reply.Value {
        name := strings.ToLower(pinfo.Name)
        description := pinfo.Description
        if strings.ToLower(description) == "<string></string>" {
            description = ""
        }
        if len( processesMap ) <= 0 || processesMap[name] {
            fmt.Printf("%-33s%-10s%s\n", name, pinfo.Statename, description)
        }
    }
}

func init() {
	parser.AddCommand("ctl",
		"Control a running daemon",
		"The ctl subcommand resembles supervisorctl command of original daemon.",
		&ctlCommand)
}
