package cmd

import (
	"context"

	cli "github.com/unstoppablemango/tdl/cli/internal"
	"github.com/unstoppablemango/tdl/pkg/pcl"
	"github.com/unstoppablemango/tdl/pkg/uml"
)

func init() {
	rootCmd.AddCommand(fromCmd)
}

var fromCmd = cli.NewFromCmd(func(ctx context.Context, co uml.ConverterOptions, args []string) (uml.Converter, error) {
	return pcl.Converter, nil
})
