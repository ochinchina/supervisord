package main

import (
	"fmt"
)

type CtlCommand struct {
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
