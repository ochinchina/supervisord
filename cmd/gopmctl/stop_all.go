package main

import (
	"context"

	"github.com/spf13/cobra"
	"github.com/stuartcarnie/gopm/rpc"
)

var stopAllCmd = cobra.Command{
	Use:   "stop-all",
	Short: "Stop all processes",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		var req rpc.StartStopAllRequest
		res, err := control.client.StopAllProcesses(context.Background(), &req)
		if err != nil {
			return err
		}
		control.printProcessInfo(res, nil)
		return nil
	},
}
