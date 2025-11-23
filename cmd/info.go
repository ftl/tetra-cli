package cmd

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ftl/tetra-cli/pkg/cli"
	"github.com/ftl/tetra-cli/pkg/radio"
)

var infoCmd = &cobra.Command{
	Use:   "info",
	Short: "Read the radio device information",
	Run:   cli.RunWithPEIAndTimeout(runInfo, fatal),
}

func init() {
	rootCmd.AddCommand(infoCmd)
}

func runInfo(ctx context.Context, pei radio.PEI, cmd *cobra.Command, args []string) {
	err := pei.ATs(ctx,
		"ATZ",
		"ATE0",
	)
	if err != nil {
		fatalf("cannot initialize radio: %v", err)
	}

	info, err := pei.AT(ctx, "ATI")
	if err != nil {
		log.Printf("cannot read radio device information: %v", err)
	} else {
		fmt.Printf("%v\n", strings.Join(info, "\n"))
	}
}
