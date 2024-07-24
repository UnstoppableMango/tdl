package cmd

import (
	"github.com/spf13/cobra"
	cli "github.com/unstoppablemango/tdl/cli/internal"
	"github.com/unstoppablemango/tdl/pkg/plugin"
	"github.com/unstoppablemango/tdl/pkg/runner"
	"github.com/unstoppablemango/tdl/pkg/uml"
)

func init() {
	genCmd.Args = cobra.MinimumNArgs(1)
	rootCmd.AddCommand(genCmd)
}

var genCmd = cli.NewGenCmd(
	func(opts uml.GeneratorOptions) (uml.Generator, error) {
		opts.Log.Debug("getting plugin name")
		source, err := plugin.ForTarget(opts.Target)
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
