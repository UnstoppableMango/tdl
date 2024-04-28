package cmd

import (
	cli "github.com/unstoppablemango/tdl/cli/internal"
	gen "github.com/unstoppablemango/tdl/pkg/go"
)

func init() {
	rootCmd.AddCommand(genCmd)
}

var genCmd = cli.NewGenCmd(gen.Go)
