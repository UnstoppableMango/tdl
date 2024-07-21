package cmd

import (
	cli "github.com/unstoppablemango/tdl/cli/internal"
	"github.com/unstoppablemango/tdl/pkg/pcl"
	"github.com/unstoppablemango/tdl/pkg/uml"
)

func init() {
	rootCmd.AddCommand(fromCmd)
}

var fromCmd = cli.NewFromCmd(
	func(opts uml.ConverterOptions) (uml.Converter, error) {
		return pcl.Converter, nil
	},
)
