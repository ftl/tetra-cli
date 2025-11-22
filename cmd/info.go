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

var infoCmd = &cobra.Command{
	Use:   "info",
	Short: "Read the radio device information",
	Run:   cli.RunWithRadioAndTimeout(runInfo, fatal),
}

func init() {
	rootCmd.AddCommand(infoCmd)
}

func runInfo(ctx context.Context, radio *com.COM, cmd *cobra.Command, args []string) {
	err := radio.ATs(ctx,
		"ATZ",
		"ATE0",
	)
	if err != nil {
		fatalf("cannot initialize radio: %v", err)
	}

	info, err := radio.AT(ctx, "ATI")
	if err != nil {
		log.Printf("cannot read radio device information: %v", err)
	} else {
		fmt.Printf("%v\n", strings.Join(info, "\n"))
	}
}
