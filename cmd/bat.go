package cmd

import (
	"context"
	"fmt"

	"github.com/ftl/tetra-pei/ctrl"
	"github.com/spf13/cobra"

	"github.com/ftl/tetra-cli/pkg/cli"
	"github.com/ftl/tetra-cli/pkg/radio"
)

var getBatteryChargeCmd = &cobra.Command{
	Use:   "bat",
	Short: "Read the current battery charge level",
	Run:   cli.RunWithPEIAndTimeout(runGetBatteryCharge, fatal),
}

func init() {
	rootCmd.AddCommand(getBatteryChargeCmd)
}

func runGetBatteryCharge(ctx context.Context, pei radio.PEI, cmd *cobra.Command, args []string) {
	err := pei.ATs(ctx,
		"ATZ",
		"ATE0",
	)
	if err != nil {
		fatalf("cannot initialize radio: %v", err)
	}

	batteryCharge, err := ctrl.RequestBatteryCharge(ctx, pei)
	if err != nil {
		fatal(err)
	}

	fmt.Printf("%d\n", batteryCharge)
}
