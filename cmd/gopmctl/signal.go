package main

import (
	"context"
	"errors"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/stuartcarnie/gopm/rpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var signalCmd = cobra.Command{
	Use:   "signal",
	Short: "Send a signal to a list of processes",
	Args:  cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		sigName, processes := args[0], args[1:]
		sigInt, ok := rpc.ProcessSignal_value[sigName]
		if !ok {
			return errors.New("invalid signal name")
		}
		req := rpc.SignalProcessRequest{Signal: rpc.ProcessSignal(sigInt)}
		for _, process := range processes {
			req.Name = process
			_, err := control.client.SignalProcess(context.Background(), &req)
			if status.Code(err) == codes.NotFound {
				fmt.Printf("Process %s not found\n", process)
			} else if err != nil {
				return err
			}
		}
		return nil
	},
}
