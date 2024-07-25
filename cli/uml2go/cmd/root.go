package cmd

import (
	"os"

	"github.com/spf13/cobra"
	cli "github.com/unstoppablemango/tdl/cli/internal"
	"github.com/unstoppablemango/tdl/pkg/logging"
)

var rootCmd = &cobra.Command{
	Use: "uml2go",
	PreRun: func(cmd *cobra.Command, args []string) {
		cli.SetLogger(cmd, logging.NewLogger())
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
