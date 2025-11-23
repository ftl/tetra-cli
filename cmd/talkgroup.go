package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/ftl/tetra-cli/pkg/cli"
	"github.com/ftl/tetra-cli/pkg/radio"
	"github.com/ftl/tetra-pei/ctrl"
	"github.com/spf13/cobra"
)

var talkgroupFlags = struct {
}{}

var setTalkgroupCmd = &cobra.Command{
	Use:   "set-talkgroup <TMO|DMO> [<GTSI>]",
	Short: "Set the operating mode and the talk group",
	Run:   cli.RunWithPEIAndTimeout(runSetTalkgroup, fatal),
}

var getTalkgroupCmd = &cobra.Command{
	Use:   "get-talkgroup",
	Short: "Get the current operating mode and the current talk group",
	Run:   cli.RunWithPEIAndTimeout(runGetTalkgroup, fatal),
}

var getTalkgroupsCmd = &cobra.Command{
	Use:   "talkgroups",
	Short: "Get all talk groups for TMO and DMO as CSV list",
	Run:   cli.RunWithPEIAndTimeout(runGetTalkgroups, fatal),
}

func init() {
	rootCmd.AddCommand(setTalkgroupCmd)
	rootCmd.AddCommand(getTalkgroupCmd)
	rootCmd.AddCommand(getTalkgroupsCmd)
}

func runSetTalkgroup(ctx context.Context, pei radio.PEI, cmd *cobra.Command, args []string) {
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

	err = pei.ATs(ctx,
		"ATZ",
		"ATE0",
		"AT+CTSP=1,1,11",
		ctrl.SetOperatingMode(aiMode),
	)
	if err != nil {
		fatalf("cannot initialize radio: %v", err)
	}

	if gtsi != "" {
		pei.AT(ctx, ctrl.SetTalkgroup(gtsi))
	}
}

func runGetTalkgroup(ctx context.Context, pei radio.PEI, cmd *cobra.Command, args []string) {
	err := pei.ATs(ctx,
		"ATZ",
		"ATE0",
		"AT+CTSP=1,1,11",
	)
	if err != nil {
		fatalf("cannot initialize radio: %v", err)
	}

	currentAIMode, err := ctrl.RequestOperatingMode(ctx, pei)
	if err != nil {
		fatalf("cannot find out the current operating mode: %v", err)
	}
	fmt.Printf("MODE: %s\n", currentAIMode)

	currentTalkgroup, err := ctrl.RequestTalkgroup(ctx, pei)
	if err != nil {
		fatalf("cannot find out the current talkgroup: %v", err)
	}
	fmt.Printf("GTSI: %s\n", currentTalkgroup)
}

func runGetTalkgroups(ctx context.Context, pei radio.PEI, cmd *cobra.Command, args []string) {
	err := pei.ATs(ctx,
		"ATZ",
		"ATE0",
	)
	if err != nil {
		fatalf("cannot initialize radio: %v", err)
	}

	lastMode, err := ctrl.RequestOperatingMode(ctx, pei)
	if err != nil {
		fatalf("cannot read last mode: %v", err)
	}

	if lastMode != ctrl.TMO {
		_, err = pei.AT(ctx, ctrl.SetOperatingMode(ctrl.TMO))
		if err != nil {
			fatalf("cannot switch to TMO: %v", err)
		}
	}
	tmoTalkgroups := make([]ctrl.TalkgroupInfo, 0, 2000)
	tmoTalkgroups, err = ctrl.RequestTalkgroups(ctx, pei, ctrl.TalkgroupDynamic, tmoTalkgroups)
	if err != nil {
		fatalf("cannot read TMO talkgroups: %v", err)
	}
	for _, info := range tmoTalkgroups {
		fmt.Printf("TMO;%s;%s\n", info.GTSI, info.Name)
	}

	_, err = pei.AT(ctx, ctrl.SetOperatingMode(ctrl.DMO))
	if err != nil {
		fatalf("cannot switch to DMO: %v", err)
	}
	dmoTalkgroups := make([]ctrl.TalkgroupInfo, 0, 2000)
	dmoTalkgroups, err = ctrl.RequestTalkgroups(ctx, pei, ctrl.TalkgroupStatic, dmoTalkgroups)
	if err != nil {
		fatalf("cannot read DMO talkgroups: %v", err)
	}
	for _, info := range dmoTalkgroups {
		fmt.Printf("DMO;%s;%s\n", info.GTSI, info.Name)
	}

	if lastMode != ctrl.DMO {
		_, err = pei.AT(ctx, ctrl.SetOperatingMode(lastMode))
		if err != nil {
			fatalf("cannot switch to last mode: %v", err)
		}
	}
}
