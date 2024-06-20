package cmd

import (
	"context"

	cli "github.com/unstoppablemango/tdl/cli/internal"
	gen "github.com/unstoppablemango/tdl/pkg/go"
	"github.com/unstoppablemango/tdl/pkg/uml"
)

func init() {
	rootCmd.AddCommand(genCmd)
}

var genCmd = cli.NewGenCmd(
	func(ctx context.Context, _ cli.GenCmdOptions, args []string) (uml.Generator, error) {
		return gen.Go, nil
	},
)
