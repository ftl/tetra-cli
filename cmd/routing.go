package cmd

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/ftl/tetra-pei/com"
	"github.com/spf13/cobra"

	"github.com/ftl/tetra-cli/pkg/cli"
)

var routingCmd = &cobra.Command{
	Use:   "routing",
	Short: "Read the current message and notification routing settings",
	Run:   cli.RunWithRadioAndTimeout(runRouting, fatal),
}

func init() {
	rootCmd.AddCommand(routingCmd)
}

func runRouting(ctx context.Context, radio *com.COM, cmd *cobra.Command, args []string) {
	err := radio.ATs(ctx,
		"ATZ",
		"ATE0",
	)
	if err != nil {
		fatalf("cannot initialize radio: %v", err)
	}

	routing, err := radio.AT(ctx, "AT+CTSP?")
	if err != nil {
		log.Printf("cannot read routing settings: %v", err)
	} else {
		fmt.Printf("%v\n", strings.Join(routing, "\n"))
	}
}
