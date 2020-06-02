package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/spf13/cobra"
)

var reloadCmd = cobra.Command{
	Use:   "reload",
	Short: "Reload the configuration for the gopm process",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		res, err := control.client.ReloadConfig(context.Background(), &empty.Empty{})
		if err != nil {
			return err
		}
		if len(res.AddedGroup) > 0 {
			fmt.Printf("Added Groups: %s\n", strings.Join(res.AddedGroup, ","))
		}
		if len(res.ChangedGroup) > 0 {
			fmt.Printf("Changed Groups: %s\n", strings.Join(res.ChangedGroup, ","))
		}
		if len(res.RemovedGroup) > 0 {
			fmt.Printf("Removed Groups: %s\n", strings.Join(res.RemovedGroup, ","))
		}
		return nil
	},
}
