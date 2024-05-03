package cmd

import (
	cli "github.com/unstoppablemango/tdl/cli/internal"
	"github.com/unstoppablemango/tdl/cli/um/runner"
)

func init() {
	rootCmd.AddCommand(toCmd)
}

var toCmd = cli.NewToCmd(&runner.Docker{})
