package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/ftl/tetra-pei/com"
	"github.com/ftl/tetra-pei/ctrl"
	"github.com/spf13/cobra"
)

var talkgroupFlags = struct {
}{}

var setTalkgroupCmd = &cobra.Command{
	Use:   "set-talkgroup <TMO|DMO> [<GTSI>]",
	Short: "Set the operating mode and the talk group",
	Run:   runCommandWithRadio(runSetTalkgroup),
}

var getTalkgroupCmd = &cobra.Command{
	Use:   "get-talkgroup",
	Short: "Get the current operating mode and the current talk group",
	Run:   runCommandWithRadio(runGetTalkgroup),
}

func init() {
	rootCmd.AddCommand(setTalkgroupCmd)
	rootCmd.AddCommand(getTalkgroupCmd)
}

func runSetTalkgroup(ctx context.Context, radio *com.COM, cmd *cobra.Command, args []string) {
	if len(args) < 1 {
		fatalf("tetra-cli set-talkgroup <TMO|DMO> [<GTSI>]")
	}

	aiMode, err := ctrl.AIModeByName(args[0])
	if err != nil {
		fatalf("invalid AI mode %s", args[0])
	}

	var gtsi string
	if len(args) > 1 {
		gtsi = strings.TrimSpace(args[1])
	}

	err = radio.ATs(ctx,
		"ATZ",
		"ATE0",
		"AT+CTSP=1,1,11",
		ctrl.SetOperatingMode(aiMode),
	)
	if err != nil {
		fatalf("cannot initialize radio: %v", err)
	}

	if gtsi != "" {
		radio.AT(ctx, ctrl.SetTalkgroup(gtsi))
	}
}

func runGetTalkgroup(ctx context.Context, radio *com.COM, cmd *cobra.Command, args []string) {
	err := radio.ATs(ctx,
		"ATZ",
		"ATE0",
		"AT+CTSP=1,1,11",
	)
	if err != nil {
		fatalf("cannot initialize radio: %v", err)
	}

	currentAIMode, err := ctrl.RequestOperatingMode(ctx, radio)
	if err != nil {
		fatalf("cannot find out the current operating mode: %v", err)
	}
	fmt.Printf("MODE: %s\n", currentAIMode)

	currentTalkgroup, err := ctrl.RequestTalkgroup(ctx, radio)
	if err != nil {
		fatalf("cannot find out the current talkgroup: %v", err)
	}
	fmt.Printf("GTSI: %s\n", currentTalkgroup)
}
