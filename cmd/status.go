package cmd

import (
	"context"

	"github.com/ftl/tetra-pei/sds"
	"github.com/ftl/tetra-pei/tetra"
	"github.com/spf13/cobra"

	"github.com/ftl/tetra-cli/pkg/cli"
	"github.com/ftl/tetra-cli/pkg/radio"
)

var statusFlags = struct {
}{}

var statusCmd = &cobra.Command{
	Use:   "status <destination ISSI> <hexstatus>",
	Short: "Send a status message",
	Run:   cli.RunWithPEIAndTimeout(runStatus, fatal),
}

func init() {
	rootCmd.AddCommand(statusCmd)
}

func runStatus(ctx context.Context, pei radio.PEI, cmd *cobra.Command, args []string) {
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

	err = pei.ATs(ctx,
		"ATZ",
		"ATE0",
		"AT+CTSP=2,2,20", // status
		sds.SwitchToStatus,
	)
	if err != nil {
		fatalf("cannot initialize radio: %v", err)
	}

	request := sds.SendMessage(destISSI, status.(sds.Encoder))
	_, err = pei.AT(ctx, request)
	if err != nil {
		fatalf("cannot send status message: %v", err)
	}
}
