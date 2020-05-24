package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/stuartcarnie/gopm/config"
	"github.com/stuartcarnie/gopm/rpc"
	"google.golang.org/grpc"
)

type Control struct {
	Configuration string
	Address       string

	client rpc.GopmClient
}

var (
	control = &Control{}

	rootCmd = cobra.Command{
		Use: "gopmctl",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return control.initializeClient()
		},
	}
)

func init() {
	rootCmd.PersistentFlags().StringVarP(&control.Configuration, "config", "c", "", "Configuration file")
	rootCmd.PersistentFlags().StringVar(&control.Address, "addr", "localhost:9002", "gopm server address")
	rootCmd.AddCommand(&statusCmd)
	rootCmd.AddCommand(&tailLogCmd)
	rootCmd.AddCommand(&signalCmd)
	rootCmd.AddCommand(&startCmd)
	rootCmd.AddCommand(&stopCmd)
	rootCmd.AddCommand(&reloadCmd)
	rootCmd.AddCommand(&shutdownCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func (ctl *Control) initializeClient() error {
	gc, err := grpc.Dial(ctl.getServerURL(), grpc.WithInsecure())
	if err != nil {
		return err
	}

	control.client = rpc.NewGopmClient(gc)
	return nil
}

func (ctl *Control) getServerURL() string {
	if ctl.Address != "" {
		return ctl.Address
	} else if _, err := os.Stat(ctl.Configuration); err == nil {
		cfg := config.NewConfig(ctl.Configuration)
		cfg.Load()
		if cfg.GrpcServer != nil && cfg.GrpcServer.Address != "" {
			return cfg.GrpcServer.Address
		}
	}
	return "localhost:9002"
}

// other commands

func (ctl *Control) printProcessInfo(res *rpc.ProcessInfoResponse, processes map[string]bool) {
	for _, pinfo := range res.Processes {
		if ctl.inProcessMap(pinfo, processes) {
			processName := pinfo.GetFullName()
			fmt.Printf("%s%-33s%-10s%s%s\n", ctl.getANSIColor(pinfo.StateName), processName, pinfo.StateName, pinfo.Description, "\x1b[0m")
		}
	}
}

func (ctl *Control) inProcessMap(procInfo *rpc.ProcessInfo, processesMap map[string]bool) bool {
	if len(processesMap) <= 0 {
		return true
	}
	for procName := range processesMap {
		if procName == procInfo.Name || procName == procInfo.GetFullName() {
			return true
		}

		// check the wildcard '*'
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

func (ctl *Control) getANSIColor(statename string) string {
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
