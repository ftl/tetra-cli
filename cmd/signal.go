package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/ftl/tetra-pei/com"
	"github.com/ftl/tetra-pei/ctrl"
	"github.com/spf13/cobra"
)

var traceSignalFlags = struct {
	scanInterval time.Duration
	scanCount    int
}{}

const defaultTraceSignalScanInterval = 30 * time.Second

var traceSignalCmd = &cobra.Command{
	Use:   "trace-signal",
	Short: "Trace the signal strength and the GPS position",
	Run:   runWithRadio(runTraceSignal),
}

func init() {
	traceSignalCmd.Flags().DurationVar(&traceSignalFlags.scanInterval, "scan-interval", defaultTraceSignalScanInterval, "scan interval")
	traceSignalCmd.Flags().IntVar(&traceSignalFlags.scanCount, "n", 0, "number of scans, 0 = infinite")

	rootCmd.AddCommand(traceSignalCmd)
}

func runTraceSignal(ctx context.Context, radio *com.COM, cmd *cobra.Command, args []string) {
	err := radio.ATs(ctx,
		"ATZ",
		"ATE0",
		"AT+CSCS=8859-1",
	)
	if err != nil {
		fatalf("cannot initilize radio: %v", err)
	}

	scanSignalAndPosition(ctx, radio)

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
				scanSignalAndPosition(ctx, radio)
				scanCount++
				if traceSignalFlags.scanCount > 0 && scanCount >= traceSignalFlags.scanCount {
					return
				}
			}
		}
	}()

	<-closed
}

func scanSignalAndPosition(ctx context.Context, radio *com.COM) {
	lat, lon, sats, timestamp, err := ctrl.RequestGPSPosition(ctx, radio)
	if err != nil {
		lat = 0
		lon = 0
		sats = 0
		timestamp = time.Now().UTC()
	}
	timeStr := timestamp.Format(time.RFC3339)

	dbm, err := ctrl.RequestSignalStrength(ctx, radio)
	if err != nil {
		dbm = 0
	}

	fmt.Printf("[%s] lat: %f lon: %f satellites: %d signal: %d dBm\n", timeStr, lat, lon, sats, dbm)
}
