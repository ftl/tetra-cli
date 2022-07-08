package cmd

import (
	"context"

	"github.com/ftl/tetra-pei/com"
	"github.com/ftl/tetra-pei/sds"
	"github.com/ftl/tetra-pei/tetra"
	"github.com/spf13/cobra"
)

var statusFlags = struct {
}{}

var statusCmd = &cobra.Command{
	Use:   "status <destination ISSI> <hexstatus>",
	Short: "Send a status message",
	Run:   runCommandWithRadio(runStatus),
}

func init() {
	rootCmd.AddCommand(statusCmd)
}

func runStatus(ctx context.Context, radio *com.COM, cmd *cobra.Command, args []string) {
	if len(args) < 2 {
		fatalf("tetra-cli status <destination ISSI> <hexstatus>")
	}

	destISSI := tetra.Identity(args[0])
	statusBytes, err := tetra.HexToBinary(args[1])
	if err != nil {
		fatalf("wrong status format: %v", err)
	}
	status, err := sds.ParseStatus(statusBytes)
	if err != nil {
		fatalf("not a valid status: %v", err)
	}

	err = radio.ATs(ctx,
		"ATZ",
		"ATE0",
		"AT+CTSP=2,2,20", // status
		sds.SwitchToStatus,
	)
	if err != nil {
		fatalf("cannot initialize radio: %v", err)
	}

	request := sds.SendMessage(destISSI, status.(sds.Encoder))
	_, err = radio.AT(ctx, request)
	if err != nil {
		fatalf("cannot send status message: %v", err)
	}
}
