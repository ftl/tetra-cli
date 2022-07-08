package cmd

import (
	"context"
	"fmt"

	"github.com/ftl/tetra-pei/com"
	"github.com/ftl/tetra-pei/ctrl"
	"github.com/spf13/cobra"
)

var getBatteryChargeCmd = &cobra.Command{
	Use:   "bat",
	Short: "Read the current battery charge level",
	Run:   runCommandWithRadio(runGetBatteryCharge),
}

func init() {
	rootCmd.AddCommand(getBatteryChargeCmd)
}

func runGetBatteryCharge(ctx context.Context, radio *com.COM, cmd *cobra.Command, args []string) {
	err := radio.ATs(ctx,
		"ATZ",
		"ATE0",
	)
	if err != nil {
		fatalf("cannot initialize radio: %v", err)
	}

	batteryCharge, err := ctrl.RequestBatteryCharge(ctx, radio)
	if err != nil {
		fatal(err)
	}

	fmt.Printf("%d\n", batteryCharge)
}
