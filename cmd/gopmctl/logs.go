package main

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/stuartcarnie/gopm/rpc"
)

type deviceType int

const (
	DeviceTypeStdout deviceType = 1 << iota
	DeviceTypeStderr

	DeviceTypeAll = DeviceTypeStdout | DeviceTypeStderr
)

func (d deviceType) String() string {
	switch d {
	case DeviceTypeStdout:
		return "stdout"
	case DeviceTypeStderr:
		return "stderr"
	case DeviceTypeAll:
		return "all"
	default:
		return "<invalid>"
	}
}

func (d *deviceType) Set(s string) error {
	switch s {
	case "stdout":
		*d = DeviceTypeStdout
	case "stderr":
		*d = DeviceTypeStderr
	case "all":
		*d = DeviceTypeAll
	default:
		return fmt.Errorf("invalid device type: %s", s)
	}

	return nil
}

func (d deviceType) Type() string {
	return "DEVICE-TYPE"
}

var tailLogOpt = struct {
	device deviceType
}{
	device: DeviceTypeStdout,
}

var tailLogCmd = cobra.Command{
	Use:   "logs",
	Short: "Fetch logs for a process",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		req := rpc.TailLogRequest{Name: args[0]}

		switch tailLogOpt.device {
		case DeviceTypeStdout:
			req.Device = rpc.LogDevice_Stdout

		case DeviceTypeStderr:
			req.Device = rpc.LogDevice_Stderr

		default:
			return fmt.Errorf("unsupported device: %s", tailLogOpt.device)
		}

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
	},
}

func init() {
	tailLogCmd.Flags().VarP(&tailLogOpt.device, "device", "d", "Device to tail (stderr|stdout)")
}
