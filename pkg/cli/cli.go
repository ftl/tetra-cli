package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/ftl/tetra-pei/com"
	"github.com/hedhyw/Go-Serial-Detector/pkg/v1/serialdet"
	"github.com/jacobsa/go-serial/serial"
	"github.com/spf13/cobra"
)

// DefaultTetraFlags defines default flags for TETRA commands:
var DefaultTetraFlags = struct {
	// Device is the filename that represents the TETRA device.
	Device string

	// CommandTimeout is the maximum duration a PEI command may take until the command is canceled.
	CommandTimeout time.Duration

	// TracePEIFilename is the name of the file used to trace the PEI communication, defined through the hidden flag "trace-pei"
	// The PEI communication is only traced, if this flag is set to a valid filename.
	TracePEIFilename string
}{}

var DefaultFatalErrorHandler func(error) = func(err error) {
	fmt.Println(err)
	os.Exit(1)
}

// InitDefaultTetraFlags adds the default TETRA flags to the given command as persistent flags.
func InitDefaultTetraFlags(command *cobra.Command, defaultCommandTimeout time.Duration) {
	command.PersistentFlags().StringVar(&DefaultTetraFlags.Device, "device", "", "serial communication device (leave empty for auto detection)")
	command.PersistentFlags().DurationVar(&DefaultTetraFlags.CommandTimeout, "commandTimeout", defaultCommandTimeout, "timeout for commands")

	// the trace-pei flag is hidden as it is mainly targeted at deveolpers
	command.PersistentFlags().StringVar(&DefaultTetraFlags.TracePEIFilename, "trace-pei", "", "filename for tracing the PEI communication")
	command.PersistentFlags().MarkHidden("trace-pei")
}

// RunWithRadioAndTimeout returns a cobra command function, that is executed using the radio defined in the "device" flag.
// Additionally, the timeout duration defined in the "commandTimeout" flag is applied.
// The fatalErrorHandler is invoked to handle any error that cannot be handled otherwise (e.g. the given device filename is invalid).
func RunWithRadioAndTimeout(run func(context.Context, *com.COM, *cobra.Command, []string), fatalErrorHandler func(error)) func(*cobra.Command, []string) {
	return RunWithRadio(func(ctx context.Context, radio *com.COM, cmd *cobra.Command, args []string) {
		cmdCtx, cancel := context.WithTimeout(ctx, DefaultTetraFlags.CommandTimeout)
		defer cancel()

		run(cmdCtx, radio, cmd, args)
	}, fatalErrorHandler)
}

// RunWithRadio returns a cobra command function, that is executed using the radio defined in the "device" flag.
// The fatalErrorHandler is invoked to handle any error that cannot be handled otherwise (e.g. the given device filename is invalid).
func RunWithRadio(run func(context.Context, *com.COM, *cobra.Command, []string), fatalErrorHandler func(error)) func(*cobra.Command, []string) {
	return func(cmd *cobra.Command, args []string) {
		if fatalErrorHandler == nil {
			fatalErrorHandler = DefaultFatalErrorHandler
		}

		portName, err := FindRadioPortName()
		if err != nil {
			fatalErrorHandler(err)
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
			fatalErrorHandler(err)
		}
		defer device.Close()

		var tracePEIFile *os.File
		if DefaultTetraFlags.TracePEIFilename != "" {
			tracePEIFile, err = os.OpenFile(DefaultTetraFlags.TracePEIFilename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				fatalErrorHandler(fmt.Errorf("cannot access PEI trace file: %v", err))
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
			fatalErrorHandler(fmt.Errorf("cannot connect to radio: %v", err))
		}

		run(rootCtx, radio, cmd, args)

		shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), DefaultTetraFlags.CommandTimeout)
		defer cancelShutdown()
		radio.AT(shutdownCtx, "ATZ")
		radio.Close()
		radio.WaitUntilClosed(shutdownCtx)
	}
}

// FindRadioPortName returns the filename for the first TETRA device it can find.
// If the device flag is set and its value is not "auto", it returns this filename.
func FindRadioPortName() (string, error) {
	if DefaultTetraFlags.Device != "" && strings.ToLower(DefaultTetraFlags.Device) != "auto" {
		return DefaultTetraFlags.Device, nil
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
