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

var genCmd = cli.NewGenCmd(func(ctx context.Context, opts uml.GeneratorOptions, args []string) (uml.Generator, error) {
	return runner.NewCli("")
})
