package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/ftl/tetra-cli/pkg/cli"
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
	cli.InitDefaultTetraFlags(rootCmd, defaultCommandTimeout)
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

func fatalf(format string, args ...any) {
	fatal(fmt.Errorf(format, args...))
}
