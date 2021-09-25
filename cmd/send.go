package cmd

import (
	"context"
	"log"
	"strings"

	"github.com/ftl/tetra-pei/com"
	"github.com/ftl/tetra-pei/sds"
	"github.com/ftl/tetra-pei/tetra"
	"github.com/spf13/cobra"
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
	Run:   runCommandWithRadio(runSend),
}

func init() {
	sendCmd.Flags().IntVar(&sendFlags.messageReference, "message-reference", 1, "the message reference used for delivery reports")
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
		"AT+CSCS=8859-1",
		sds.SwitchToSDSTL,
	)

	messageReceived := make(chan struct{})
	messageConsumed := make(chan struct{})
	var decodeMessagePart = func(lines []string) {
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
	}
	radio.AddIndication("+CTSDSR: 12,", 1, decodeMessagePart)

	maxPDUBits, err := sds.RequestMaxMessagePDUBits(ctx, radio.AT)
	if err != nil {
		fatalf("cannot find out how long an SDS text message may be: %v", err)
	}

	var pdu sds.Encoder
	var sdsTransfer sds.SDSTransfer
	if sendFlags.simple {
		pdu = sds.NewSimpleTextMessage(sendFlags.immediate, encoding, messageText)
	} else {
		sdsTransfer = sds.NewTextMessageTransfer(messageReference, sendFlags.immediate, deliveryReport, encoding, messageText)
		pdu = sdsTransfer
	}
	_, pduBits := pdu.Encode([]byte{}, 0)
	if pduBits > maxPDUBits {
		fatalf("the message is too long: expected max %d bits, but got %d", maxPDUBits, pduBits)
	}

	request := sds.SendMessage(destISSI, pdu)
	_, err = radio.AT(ctx, request)
	if err != nil {
		fatalf("cannot send SDS text message: %v", err)
	}

	if deliveryReport == sds.NoReportRequested {
		return
	}
	if sdsTransfer.ReceivedReportRequested() {
		select {
		case <-messageReceived:
			log.Printf("message received")
		case <-ctx.Done():
			fatal(ctx.Err())
		}
	}
	if sdsTransfer.ConsumedReportRequested() {
		select {
		case <-messageConsumed:
			log.Printf("message consumed")
		case <-ctx.Done():
			fatal(ctx.Err())
		}
	}
}
