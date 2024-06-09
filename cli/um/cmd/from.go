package cmd

import (
	"context"

	"github.com/spf13/cobra"
	cli "github.com/unstoppablemango/tdl/cli/internal"
	"github.com/unstoppablemango/tdl/pkg/runner"
	"github.com/unstoppablemango/tdl/pkg/uml"
)

func init() {
	fromCmd.Args = cobra.ExactArgs(1)
	rootCmd.AddCommand(fromCmd)
}

var fromCmd = cli.NewFromCmd(func(ctx context.Context, opts uml.ConverterOptions, args []string) (uml.Converter, error) {
	return runner.NewCli("")
})
