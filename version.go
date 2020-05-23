package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

// VERSION the version of supervisor
const VERSION = "v0.6.8"

var versionCommand = cobra.Command{
	Use: "version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(VERSION)
	},
}

func init() {
	rootCmd.AddCommand(&versionCommand)
}
