package main

import (
	"context"

	"github.com/spf13/cobra"
	"github.com/stuartcarnie/gopm/rpc"
)

var startAllCmd = cobra.Command{
	Use:   "start-all",
	Short: "Start all processes",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		var req rpc.StartStopAllRequest
		res, err := control.client.StartAllProcesses(context.Background(), &req)
		if err != nil {
			return err
		}
		control.printProcessInfo(res, nil)
		return nil
	},
}
