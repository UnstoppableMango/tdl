package cmd

import (
	"os"

	"github.com/spf13/cobra"
	cli "github.com/unstoppablemango/tdl/cli/internal"
)

var rootCmd = &cobra.Command{
	Use: "uml2go",
	PreRun: func(cmd *cobra.Command, args []string) {
		cmd.SetContext(cli.WithLogger(cmd.Context()))
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
