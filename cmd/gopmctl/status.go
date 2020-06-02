package main

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/spf13/cobra"
)

var statusCmd = cobra.Command{
	Use:   "status",
	Short: "Display the status of a list of processes",
	RunE: func(cmd *cobra.Command, args []string) error {
		res, err := control.client.GetProcessInfo(context.Background(), &empty.Empty{})
		if err != nil {
			return err
		}
		processesMap := make(map[string]bool)
		for _, process := range args {
			processesMap[process] = true
		}
		control.printProcessInfo(res, processesMap)
		return nil
	},
}
