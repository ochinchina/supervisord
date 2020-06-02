package main

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/stuartcarnie/gopm/rpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var stopCmd = cobra.Command{
	Use:   "stop process...",
	Short: "Stop a list of processes",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		for _, name := range args {
			req := rpc.StartStopRequest{Name: name, Wait: true}
			_, err := control.client.StopProcess(context.Background(), &req)
			if status.Code(err) == codes.NotFound {
				fmt.Printf("Process not found: %s\n", name)
			} else if err != nil {
				return err
			}
			fmt.Printf("Process stopped: %s\n", name)
		}
		return nil
	},
}
