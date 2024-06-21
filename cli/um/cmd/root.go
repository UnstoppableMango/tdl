package cmd

import (
	"os"

	"github.com/spf13/cobra"
	cli "github.com/unstoppablemango/tdl/cli/internal"
)

var (
	inputFile string
	rootCmd   = &cobra.Command{
		Use:   "um",
		Short: "UnstoppableMango's Type Description Language CLI",
		PreRun: func(cmd *cobra.Command, args []string) {
			cmd.SetContext(cli.WithLogger(cmd.Context()))
		},
	}
)

func Execute() {
	cli.InputFile(rootCmd, &inputFile, "Path to the input file")
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
