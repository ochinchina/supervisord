package main

import (
	"fmt"
)

type CtlCommand struct {
	Host string `short:"h" long:"host" description:"host on which supervisord server is running." default:"localhost"`
	Port int    `short:"p" long:"port" description:"port which supervisord server is listening." default:"9001"`
}

var ctlCommand CtlCommand

func (x *CtlCommand) Execute(args []string) error {
	fmt.Printf("empty ctl command")
	return nil
}

func init() {
	parser.AddCommand("ctl",
		"Control a running daemon",
		"The ctl subcommand resembles supervisorctl command of original daemon.",
		&ctlCommand)
}
