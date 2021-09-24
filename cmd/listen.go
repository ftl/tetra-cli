package cmd

import (
	"context"
	"fmt"
	"log"

	"github.com/ftl/tetra-pei/com"
	"github.com/ftl/tetra-pei/sds"
	"github.com/spf13/cobra"
)

var listenFlags = struct {
}{}

var listenCmd = &cobra.Command{
	Use:   "listen",
	Short: "Listen for incoming text and status messages",
	Run:   runWithRadio(runListen),
}

func init() {
	rootCmd.AddCommand(listenCmd)
}

func runListen(ctx context.Context, radio *com.COM, cmd *cobra.Command, args []string) {
	err := radio.ATs(ctx,
		"ATZ",
		"AT+CSCS=8859-1",
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
		fmt.Println(m)
	}).WithResponseCallback(func(responses []string) error {
		for _, response := range responses {
			_, err := radio.AT(ctx, response)
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

	radio.AddIndication("+CTSDSR: 12,", 1, decodeMessagePart)
	radio.AddIndication("+CTSDSR: 13,", 1, decodeMessagePart)
	radio.AddIndication("+ENCR", 0, func(lines []string) {
		fmt.Printf("VOICE\n%s\n--\n", lines[0])
	})

	<-ctx.Done()
}