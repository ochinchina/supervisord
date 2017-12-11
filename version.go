package main

import (
	"fmt"
)

const VERSION = "1.0.005"

type VersionCommand struct {
}

var versionCommand VersionCommand

func (v VersionCommand) Execute(args []string) error {
	fmt.Println(VERSION)
	return nil
}

func init() {
	parser.AddCommand("version",
		"show the version of supervisor",
		"display the supervisor version",
		&versionCommand)
}
