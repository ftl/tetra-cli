package cmd

import (
	"fmt"
	"log"

	"github.com/hedhyw/Go-Serial-Detector/pkg/v1/serialdet"
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
	devices, err := serialdet.List()
	if err != nil {
		log.Fatal(err)
	}

	if len(devices) == 0 {
		fmt.Printf("no active serial devices found\n")
		return
	}

	for _, device := range devices {
		fmt.Printf("%s: %s\n", device.Description(), device.Path())
	}
}
