package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/chmorgan/go-serial2/serial"
	"github.com/ftl/tetra-pei/com"
	"github.com/hedhyw/Go-Serial-Detector/pkg/v1/serialdet"
	"github.com/spf13/cobra"
)

var rootFlags = struct {
	device           string
	commandTimeout   time.Duration
	tracePEIFilename string
}{}

const defaultCommandTimeout = 5 * time.Second

var rootCmd = &cobra.Command{
	Use:   "tetra-cli",
	Short: "Control a TETRA radio terminal through its PEI.",
}

func init() {
	rootCmd.PersistentFlags().StringVar(&rootFlags.device, "device", "", "serial communication device (leave empty for auto detection)")
	rootCmd.PersistentFlags().DurationVar(&rootFlags.commandTimeout, "commandTimeout", defaultCommandTimeout, "timeout for commands")
	rootCmd.PersistentFlags().StringVar(&rootFlags.tracePEIFilename, "trace-pei", "", "filename for tracing the PEI communication")
	rootCmd.PersistentFlags().MarkHidden("trace-pei")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fatal(err)
	}
}

func fatal(err error) {
	fmt.Println(err)
	os.Exit(1)
}

func fatalf(format string, args ...interface{}) {
	fatal(fmt.Errorf(format, args...))
}

func runCommandWithRadio(run func(context.Context, *com.COM, *cobra.Command, []string)) func(*cobra.Command, []string) {
	return runWithRadio(func(ctx context.Context, radio *com.COM, cmd *cobra.Command, args []string) {
		cmdCtx, cancel := context.WithTimeout(ctx, rootFlags.commandTimeout)
		defer cancel()

		run(cmdCtx, radio, cmd, args)
	})
}

func runWithRadio(run func(context.Context, *com.COM, *cobra.Command, []string)) func(*cobra.Command, []string) {
	return func(cmd *cobra.Command, args []string) {
		portName, err := getRadioPortName()
		if err != nil {
			fatal(err)
		}

		portConfig := serial.OpenOptions{
			PortName:              portName,
			BaudRate:              38400,
			DataBits:              8,
			StopBits:              1,
			ParityMode:            serial.PARITY_NONE,
			RTSCTSFlowControl:     true,
			MinimumReadSize:       4,
			InterCharacterTimeout: 100,
		}
		device, err := serial.Open(portConfig)
		if err != nil {
			fatal(err)
		}
		defer device.Close()

		var tracePEIFile *os.File
		if rootFlags.tracePEIFilename != "" {
			tracePEIFile, err = os.OpenFile(rootFlags.tracePEIFilename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				fatalf("cannot access PEI trace file: %v", err)
			}
			defer tracePEIFile.Close()
		}

		rootCtx, interrupted := signal.NotifyContext(context.Background(), os.Interrupt)
		defer interrupted()

		var radio *com.COM
		if tracePEIFile != nil {
			radio = com.NewWithTrace(device, tracePEIFile)
		} else {
			radio = com.New(device)
		}
		err = radio.ClearSyntaxErrors(rootCtx)
		if err != nil {
			fatalf("cannot connect to radio: %v", err)
		}

		run(rootCtx, radio, cmd, args)

		shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), rootFlags.commandTimeout)
		defer cancelShutdown()
		radio.AT(shutdownCtx, "ATZ")
		radio.Close()
		radio.WaitUntilClosed(shutdownCtx)
	}
}

func getRadioPortName() (string, error) {
	if rootFlags.device != "" && strings.ToLower(rootFlags.device) != "auto" {
		return rootFlags.device, nil
	}

	devices, err := serialdet.List()
	if err != nil {
		return "", err
	}

	for _, device := range devices {
		description := strings.ToLower(device.Description())
		if strings.Contains(description, "tetra_pei_interface") {
			return device.Path(), nil
		}
	}

	return "", fmt.Errorf("no active PEI interface found, use the --device parameter to provide the serial communication device")
}
