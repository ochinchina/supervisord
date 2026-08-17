package main

import (
	"fmt"
)

// VERSION the version of supervisor

var (
	version = "v0.7.3"
	commit  = ""
)

// VersionCommand implement the flags.Commander interface
type VersionCommand struct {
}

var versionCommand VersionCommand

// Execute implement Execute() method defined in flags.Commander interface, executes the given command
func (v VersionCommand) Execute(args []string) error {
	fmt.Println("Version:", version)
	fmt.Println(" Commit:", commit)
	return nil
}

func init() {
	parser.AddCommand("version",
		"show the version of supervisor",
		"display the supervisor version",
		&versionCommand)
}
