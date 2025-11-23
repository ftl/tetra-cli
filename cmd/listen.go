package cmd

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/ftl/tetra-pei/ctrl"
	"github.com/ftl/tetra-pei/sds"
	"github.com/spf13/cobra"

	"github.com/ftl/tetra-cli/pkg/cli"
	"github.com/ftl/tetra-cli/pkg/radio"
)

var listenFlags = struct {
}{}

var listenCmd = &cobra.Command{
	Use:   "listen",
	Short: "Listen for incoming text and status messages",
	Run:   cli.RunWithPEI(runListen, fatal),
}

func init() {
	rootCmd.AddCommand(listenCmd)
}

func runListen(ctx context.Context, pei radio.PEI, cmd *cobra.Command, args []string) {
	err := pei.ATs(ctx,
		"ATZ",
		"ATE0",
		"AT+CSCS=8859-1",
		"AT+CTSP=2,0,0",   // call signaling
		"AT+CTSP=2,2,20",  // status
		"AT+CTSP=1,3,2",   // simple text messaging
		"AT+CTSP=1,3,9",   // simple immediate text messaging
		"AT+CTSP=1,3,130", // text messaging
		"AT+CTSP=1,3,137", // immediate text messaging
		"AT+CTSP=1,3,138", // message with UDH
	)
	if err != nil {
		fatalf("cannot initialize radio: %v", err)
	}

	stack := sds.NewStack().WithMessageCallback(func(m sds.Message) {
		var opta, sanitizedText, itsi string
		opta, sanitizedText = sds.SplitLeadingOPTA(m.Text())
		sanitizedText, itsi = sds.SplitTrailingITSI(sanitizedText)
		fmt.Printf("MESSAGE\nISSI:%s\n", m.Source)
		if itsi != "" {
			fmt.Printf("ITSI:%s\n", itsi)
		}
		if opta != "" {
			fmt.Printf("OPTA:%s\n", opta)
		}
		fmt.Printf("TEXT:%s\n", sanitizedText)
		fmt.Println("--")
	}).WithStatusCallback(func(m sds.StatusMessage) {
		fmt.Printf("STATUS\nISSI:%s\nSTATUS:%4x\n--\n", m.Source, m.Value)
	}).WithResponseCallback(func(responses []string) error {
		for _, response := range responses {
			_, err := pei.AT(ctx, response)
			if err != nil {
				log.Printf("cannot send response command %s:\n%v", response, err)
				return err
			}
		}
		return nil
	})

	var decodeMessagePart = func(lines []string) {
		if len(lines) == 2 {
			part, err := sds.ParseIncomingMessage(lines[0], lines[1])
			if err != nil {
				log.Printf("cannot decode message part: %v", err)
				return
			}
			stack.Put(part)
		}
	}

	pei.AddIndication("+CTSDSR: 12,", 1, decodeMessagePart)
	pei.AddIndication("+CTSDSR: 13,", 1, decodeMessagePart)
	pei.AddIndication("+CTXG:", 0, func(lines []string) {
		parts := strings.Split(lines[0][6:], ",")
		switch len(parts) {
		case 4:
			fmt.Print("VOICE TX\n--\n")
		case 6:
			fmt.Printf("VOICE RX\nITSI: %s\n--\n", parts[5])
		}
	})
	pei.AddIndication("+CDTXC:", 0, func(lines []string) {
		fmt.Printf("TALKGROUP IDLE\n--\n")
	})
	pei.AddIndication("+CTCR:", 0, func(lines []string) {
		fmt.Printf("TALKGROUP INACTIVE\n--\n")
	})
	pei.AddIndication("+CTOM: ", 0, func(lines []string) {
		aiMode, err := strconv.Atoi(lines[0][7:])
		if err != nil {
			return
		}
		fmt.Printf("AI MODE: %s\n--\n", ctrl.AIMode(aiMode).String())
	})

	<-ctx.Done()
}
