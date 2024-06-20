package cmd

import (
	"context"

	"github.com/spf13/cobra"
	cli "github.com/unstoppablemango/tdl/cli/internal"
	"github.com/unstoppablemango/tdl/pkg/runner"
	"github.com/unstoppablemango/tdl/pkg/uml"
)

func init() {
	genCmd.Args = cobra.ExactArgs(1)
	rootCmd.AddCommand(genCmd)
}

var genCmd = cli.NewGenCmd(
	func(ctx context.Context, opts cli.GenCmdOptions, args []string) (uml.Generator, error) {
		opts.Log.Debug("executing cli runner")
		return runner.NewCli("")
	},
)
