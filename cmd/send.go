package cmd

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"strings"

	"github.com/ftl/tetra-pei/com"
	"github.com/ftl/tetra-pei/sds"
	"github.com/ftl/tetra-pei/tetra"
	"github.com/spf13/cobra"

	"github.com/ftl/tetra-cli/pkg/cli"
)

var sendFlags = struct {
	messageReference int
	immediate        bool
	ackReceive       bool
	ackConsume       bool
	simple           bool
	encoding         string
}{}

var sendCmd = &cobra.Command{
	Use:   "send <destination ISSI> <text>",
	Short: "Send an SDS text message",
	Run:   cli.RunWithRadioAndTimeout(runSend, fatal),
}

func init() {
	sendCmd.Flags().IntVar(&sendFlags.messageReference, "message-reference", 0, "the message reference used for delivery reports")
	sendCmd.Flags().BoolVar(&sendFlags.immediate, "immediate", false, "immediately show the message at the receiver")
	sendCmd.Flags().BoolVar(&sendFlags.ackReceive, "ack-receive", false, "request acknowledgment for receiving the message")
	sendCmd.Flags().BoolVar(&sendFlags.ackConsume, "ack-consume", false, "request acknowledgment for consuming the message")
	sendCmd.Flags().BoolVar(&sendFlags.simple, "simple", false, "use the simple text messaging protocol (no delivery reports possible)")
	sendCmd.Flags().StringVar(&sendFlags.encoding, "encoding", "ISO8859-1", "the text encoding")

	rootCmd.AddCommand(sendCmd)
}

func runSend(ctx context.Context, radio *com.COM, cmd *cobra.Command, args []string) {
	if len(args) < 2 {
		fatalf("tetra-cli send <destination ISSI> <text>")
	}

	destISSI := tetra.Identity(args[0])
	if sendFlags.messageReference == 0 {
		sendFlags.messageReference = (rand.Int() + 1) & 0xFF
	}
	if sendFlags.messageReference < 1 || sendFlags.messageReference > 255 {
		fatalf("the message reference must be 1-255, but got %d", sendFlags.messageReference)
	}
	messageReference := sds.MessageReference(sendFlags.messageReference)

	encoding, ok := sds.EncodingByName[strings.ToUpper(strings.TrimSpace(sendFlags.encoding))]
	if !ok {
		fatalf("unexpected encoding: %s", sendFlags.encoding)
	}
	messageText := strings.Join(args[1:], " ")
	deliveryReport := sds.NoReportRequested
	if sendFlags.ackReceive {
		deliveryReport |= sds.MessageReceivedReportRequested
	}
	if sendFlags.ackConsume {
		deliveryReport |= sds.MessageConsumedReportRequested
	}

	err := radio.ATs(ctx,
		"ATZ",
		"ATE0",
		"AT+CSCS=8859-1",
		sds.SwitchToSDSTL,
	)
	if err != nil {
		fatalf("cannot initialize radio: %v", err)
	}

	maxPDUBits, err := sds.RequestMaxMessagePDUBits(ctx, radio)
	if err != nil {
		fatalf("cannot find out how long an SDS text message may be: %v", err)
	}
	maxPDUBits = 668 // this is the value that seems to work in practice. grateful for any hint how this is supposed to work

	var pdu sds.Encoder
	var sdsTransfer sds.SDSTransfer
	if sendFlags.simple {
		pdu = sds.NewSimpleTextMessage(sendFlags.immediate, encoding, messageText)
	} else {
		sdsTransfer = sds.NewTextMessageTransfer(messageReference, sendFlags.immediate, deliveryReport, encoding, messageText)
		pdu = sdsTransfer
	}

	_, pduBits := pdu.Encode([]byte{}, 0)
	if pduBits <= maxPDUBits {
		err = sendSingleTextMessage(ctx, radio, destISSI, messageReference, pdu, sdsTransfer.ReceivedReportRequested(), sdsTransfer.ConsumedReportRequested())
	} else {
		err = sendConcatenatedTextMessage(ctx, radio, destISSI, messageReference, encoding, maxPDUBits, messageText)
	}

	if err != nil {
		fatal(err)
	}
}

func sendSingleTextMessage(ctx context.Context, radio *com.COM, destISSI tetra.Identity, messageReference sds.MessageReference, pdu sds.Encoder, waitForReceived bool, waitForConsumed bool) error {
	messageReceived := make(chan struct{})
	messageConsumed := make(chan struct{})
	radio.AddIndication("+CTSDSR: 12,", 1, func(lines []string) {
		if len(lines) == 2 {
			part, err := sds.ParseIncomingMessage(lines[0], lines[1])
			if err != nil {
				log.Printf("cannot decode message part: %v", err)
			}

			switch report := part.Payload.(type) {
			case sds.SDSReport:
				if report.MessageReference != messageReference {
					return
				}
				switch report.DeliveryStatus {
				case sds.ReceiptAckByDestination:
					close(messageReceived)
				case sds.ConsumedByDestination:
					close(messageConsumed)
				default:
					log.Printf("unexpected delivery report: 0x%x", report.DeliveryStatus)
				}

			case sds.SDSShortReport:
				if report.MessageReference != messageReference {
					return
				}
				switch report.ReportType {
				case sds.MessageReceivedShort:
					close(messageReceived)
				case sds.MessageConsumedShort:
					close(messageConsumed)
				default:
					log.Printf("unexpected short delivery report: 0x%x", report.ReportType)
				}
			}
		}
	})

	request := sds.SendMessage(destISSI, pdu)
	_, err := radio.AT(ctx, request)
	if err != nil {
		return fmt.Errorf("cannot send SDS text message: %v", err)
	}

	if waitForReceived {
		select {
		case <-messageReceived:
			log.Printf("message received")
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	if waitForConsumed {
		select {
		case <-messageConsumed:
			log.Printf("message consumed")
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return nil
}

func sendConcatenatedTextMessage(ctx context.Context, radio *com.COM, destISSI tetra.Identity, messageReference sds.MessageReference, encoding sds.TextEncoding, maxPDUBits int, messageText string) error {
	partConfirmation := make(chan string, 1)
	radio.AddIndication("+CMGS: 0,", 0, func(lines []string) {
		if len(lines) != 1 {
			return
		}
		partConfirmation <- lines[0]
	})

	pdus := sds.NewConcatenatedMessageTransfer(messageReference, sds.NoReportRequested, encoding, maxPDUBits, messageText)
	for i, pdu := range pdus {
		request := sds.SendMessage(destISSI, pdu)
		_, err := radio.AT(ctx, request)
		if err != nil {
			return fmt.Errorf("cannot send SDS text message part #%d: %v", i+1, err)
		}
		if i < len(pdus)-1 {
			select {
			case <-partConfirmation:
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}
	return nil
}
