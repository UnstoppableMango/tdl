package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	cli "github.com/unstoppablemango/tdl/cli/internal"
)

var rootCmd = &cobra.Command{
	Use: "uml2pcl",
	PreRun: func(cmd *cobra.Command, args []string) {
		cmd.SetContext(cli.WithLogger(cmd.Context()))
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
