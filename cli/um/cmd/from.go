package cmd

import (
	"context"

	"github.com/spf13/cobra"
	cli "github.com/unstoppablemango/tdl/cli/internal"
	"github.com/unstoppablemango/tdl/pkg/plugin"
	"github.com/unstoppablemango/tdl/pkg/runner"
	"github.com/unstoppablemango/tdl/pkg/uml"
)

func init() {
	fromCmd.Args = cobra.MinimumNArgs(1)
	rootCmd.AddCommand(fromCmd)
}

var fromCmd = cli.NewFromCmd(
	func(ctx context.Context, opts cli.FromCmdOptions, args []string) (uml.Converter, error) {
		opts.Log.Debug("getting plugin name")
		source, err := plugin.ForTarget(args[0])
		if err != nil {
			return nil, err
		}

		opts.Log.Debug("looking up plugin")
		bin, err := plugin.LookupPath(source)
		if err != nil {
			return nil, err
		}

		opts.Log.Debug("executing cli runner")
		return runner.NewCli(bin,
			runner.WithLogger(opts.Log),
		)
	},
)
