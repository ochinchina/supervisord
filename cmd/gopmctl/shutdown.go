package main

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/spf13/cobra"
)

var shutdownCmd = cobra.Command{
	Use:   "shutdown",
	Short: "Shutdown the gopm service",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		_, err := control.client.Shutdown(context.Background(), &empty.Empty{})
		if err != nil {
			return err
		}
		return nil
	},
}
