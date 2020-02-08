package main

import (
	"fmt"
)

//Version is minor version of supervisord
const Version = "v0.6"

//VersionCommand is interface to receive version read request
type VersionCommand struct {
}

var versionCommand VersionCommand

// Execute executes the get-version command
func (v VersionCommand) Execute(args []string) error {
	fmt.Println(Version)
	return nil
}

func init() {
	parser.AddCommand("version",
		"show the version of supervisor",
		"display the supervisor version",
		&versionCommand)
}
