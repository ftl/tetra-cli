package cmd

import (
	"fmt"

	"github.com/ftl/tetra-pei/serial"
	"github.com/spf13/cobra"
)

var listDevicesCmd = &cobra.Command{
	Use:   "list_devices",
	Short: "List all active serial devices",
	Run:   runListDevices,
}

func init() {
	rootCmd.AddCommand(listDevicesCmd)
}

func runListDevices(*cobra.Command, []string) {
	devices, err := serial.ListDevices()
	if err != nil {
		fatal(err)
	}

	if len(devices) == 0 {
		fmt.Printf("no active serial devices found\n")
		return
	}

	for _, device := range devices {
		fmt.Printf("%s: %s\n", device.Description, device.Filename)
	}
}
