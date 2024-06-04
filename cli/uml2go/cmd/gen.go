package cmd

import (
	cli "github.com/unstoppablemango/tdl/cli/internal"
	gen "github.com/unstoppablemango/tdl/pkg/go"
	"github.com/unstoppablemango/tdl/pkg/uml"
)

func init() {
	rootCmd.AddCommand(genCmd)
}

var genCmd = cli.NewGenCmd(func(_ uml.GeneratorOptions) uml.Generator {
	return gen.Go
})
