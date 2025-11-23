package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/ftl/tetra-pei/ctrl"
	"github.com/spf13/cobra"

	"github.com/ftl/tetra-cli/pkg/cli"
	"github.com/ftl/tetra-cli/pkg/radio"
)

var traceSignalFlags = struct {
	scanInterval time.Duration
	scanCount    int
}{}

const defaultTraceSignalScanInterval = 30 * time.Second

var traceSignalCmd = &cobra.Command{
	Use:   "trace-signal",
	Short: "Trace the signal strength and the GPS position",
	Run:   cli.RunWithPEI(runTraceSignal, fatal),
}

func init() {
	traceSignalCmd.Flags().DurationVar(&traceSignalFlags.scanInterval, "scan-interval", defaultTraceSignalScanInterval, "scan interval")
	traceSignalCmd.Flags().IntVar(&traceSignalFlags.scanCount, "n", 0, "number of scans, 0 = infinite")

	rootCmd.AddCommand(traceSignalCmd)
}

func runTraceSignal(ctx context.Context, pei radio.PEI, cmd *cobra.Command, args []string) {
	err := pei.ATs(ctx,
		"ATZ",
		"ATE0",
		"AT+CSCS=8859-1",
	)
	if err != nil {
		fatalf("cannot initilize radio: %v", err)
	}

	scanSignalAndPosition(ctx, pei)

	if traceSignalFlags.scanCount == 1 {
		return
	}

	closed := make(chan struct{})
	go func() {
		defer close(closed)

		scanTicker := time.NewTicker(traceSignalFlags.scanInterval)
		defer scanTicker.Stop()

		scanCount := 1
		for {
			select {
			case <-ctx.Done():
				return
			case <-scanTicker.C:
				scanSignalAndPosition(ctx, pei)
				scanCount++
				if traceSignalFlags.scanCount > 0 && scanCount >= traceSignalFlags.scanCount {
					return
				}
			}
		}
	}()

	<-closed
}

func scanSignalAndPosition(ctx context.Context, pei radio.PEI) {
	lat, lon, sats, timestamp, err := ctrl.RequestGPSPosition(ctx, pei)
	if err != nil {
		lat = 0
		lon = 0
		sats = 0
		timestamp = time.Now().UTC()
	}
	timeStr := timestamp.Format(time.RFC3339)

	dbm, err := ctrl.RequestSignalStrength(ctx, pei)
	if err != nil {
		dbm = 0
	}

	fmt.Printf("[%s] lat: %f lon: %f satellites: %d signal: %d dBm\n", timeStr, lat, lon, sats, dbm)
}
