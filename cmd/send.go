package cmd

import (
	"context"
	"strings"

	"github.com/ftl/tetra-pei/com"
	"github.com/ftl/tetra-pei/sds"
	"github.com/ftl/tetra-pei/tetra"
	"github.com/spf13/cobra"
)

var sendFlags = struct {
}{}

var sendCmd = &cobra.Command{
	Use:   "send <destination ISSI> <text>",
	Short: "Send an SDS text message",
	Run:   runCommandWithRadio(runSend),
}

func init() {
	rootCmd.AddCommand(sendCmd)
}

func runSend(ctx context.Context, radio *com.COM, cmd *cobra.Command, args []string) {
	if len(args) < 2 {
		fatalf("tetra-cli send <destination ISSI> <text>")
	}

	destISSI := tetra.Identity(args[0])
	messageText := strings.Join(args[1:], " ")

	err := radio.ATs(ctx,
		"ATZ",
		"AT+CSCS=8859-1",
		sds.SwitchToSDSTL,
	)
	if err != nil {
		fatalf("cannot initialize radio: %v", err)
	}

	maxPDUBits, err := sds.RequestMaxMessagePDUBits(ctx, radio.AT)
	if err != nil {
		fatalf("cannot find out how long an SDS text message may be: %v", err)
	}

	sdsTransfer := sds.NewTextMessageTransfer(0x01, messageText)
	_, pduBits := sdsTransfer.Encode([]byte{}, 0)
	if pduBits > maxPDUBits {
		fatalf("the message is too long: expected max %d bits, but got %d", maxPDUBits, pduBits)
	}

	request := sds.SendMessage(destISSI, sdsTransfer)
	_, err = radio.AT(ctx, request)
	if err != nil {
		fatalf("cannot send SDS text message: %v", err)
	}
}
