package cmd

import (
	cli "github.com/unstoppablemango/tdl/cli/internal"
	"github.com/unstoppablemango/tdl/cli/um/runner"
)

func init() {
	rootCmd.AddCommand(fromCmd)
}

var fromCmd = cli.NewFromCmd(&runner.Docker{})
