package main

import (
	"context"
	"os"

	"github.com/spf13/cobra"
	"github.com/stuartcarnie/gopm/rpc"
)

var tailLogCmd = cobra.Command{Use: "logs", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
	req := rpc.TailLogRequest{Name: args[0], Device: rpc.LogDevice_Stderr}
	stream, err := control.client.TailLog(context.Background(), &req)
	if err != nil {
		return err
	}
	msg := new(rpc.TailLogResponse)
	for {
		err := stream.RecvMsg(msg)
		if err != nil {
			break
		}
		_, _ = os.Stdout.Write(msg.Lines)
		msg.Lines = msg.Lines[:0]
	}
	return nil
}}
