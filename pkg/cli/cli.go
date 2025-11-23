package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/ftl/tetra-pei/serial"
	"github.com/spf13/cobra"

	"github.com/ftl/tetra-cli/pkg/radio"
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

// RunWithPEIAndTimeout returns a cobra command function, that is executed using the PEI device defined in the "device" flag.
// Additionally, the timeout duration defined in the "commandTimeout" flag is applied.
// The fatalErrorHandler is invoked to handle any error that cannot be handled otherwise (e.g. the given device filename is invalid).
func RunWithPEIAndTimeout(run func(context.Context, radio.PEI, *cobra.Command, []string), fatalErrorHandler func(error)) func(*cobra.Command, []string) {
	return RunWithPEI(func(ctx context.Context, pei radio.PEI, cmd *cobra.Command, args []string) {
		cmdCtx, cancel := context.WithTimeout(ctx, DefaultTetraFlags.CommandTimeout)
		defer cancel()

		run(cmdCtx, pei, cmd, args)
	}, fatalErrorHandler)
}

// RunWithPEI returns a cobra command function, that is executed using the PEI device defined in the "device" flag.
// The fatalErrorHandler is invoked to handle any error that cannot be handled otherwise (e.g. the given device filename is invalid).
func RunWithPEI(run func(context.Context, radio.PEI, *cobra.Command, []string), fatalErrorHandler func(error)) func(*cobra.Command, []string) {
	return func(cmd *cobra.Command, args []string) {
		if fatalErrorHandler == nil {
			fatalErrorHandler = DefaultFatalErrorHandler
		}

		var err error
		rootCtx := cmd.Context()

		tracePEIFile, err := setupTracePEI()
		if err != nil {
			fatalErrorHandler(fmt.Errorf("cannot access PEI trace file: %v", err))
		}

		portName, err := FindRadioPortName()
		if err != nil {
			fatalErrorHandler(err)
		}

		var pei radio.PEI
		if tracePEIFile != nil {
			defer tracePEIFile.Close()
			pei, err = serial.OpenWithTrace(portName, tracePEIFile)
		} else {
			pei, err = serial.Open(portName)
		}
		if err != nil {
			fatalErrorHandler(fmt.Errorf("cannot connect to radio: %v", err))
		}

		err = pei.ClearSyntaxErrors(rootCtx)
		if err != nil {
			fatalErrorHandler(fmt.Errorf("cannot initialize radio: %v", err))
		}

		run(rootCtx, pei, cmd, args)

		shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), DefaultTetraFlags.CommandTimeout)
		defer cancelShutdown()
		pei.AT(shutdownCtx, "ATZ")
		pei.Close()
		pei.WaitUntilClosed(shutdownCtx)
	}
}

func setupTracePEI() (io.WriteCloser, error) {
	if DefaultTetraFlags.TracePEIFilename == "" {
		return nil, nil
	}

	return os.OpenFile(DefaultTetraFlags.TracePEIFilename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
}

// FindRadioPortName returns the filename for the first TETRA device it can find.
// If the device flag is set and its value is not "auto", it returns this filename.
func FindRadioPortName() (string, error) {
	if DefaultTetraFlags.Device != "" && strings.ToLower(DefaultTetraFlags.Device) != "auto" {
		return DefaultTetraFlags.Device, nil
	}

	portName, err := serial.FindRadioPortName()
	if errors.Is(err, serial.NoPEIFound) {
		return "", fmt.Errorf("no active PEI interface found, use the --device parameter to provide the serial communication device")
	}
	if err != nil {
		return "", err
	}
	return portName, nil
}
