package main

import (
	"github.com/unstoppablemango/tdl/internal/util"
	"github.com/unstoppablemango/tdl/pkg/cmd"
)

func main() {
	rootCmd := cmd.NewUx()
	rootCmd.AddCommand(
		cmd.NewGen(),
		cmd.NewPlugin(),
		cmd.NewTesting(),
		cmd.NewWhich(),
	)

	if err := rootCmd.Execute(); err != nil {
		util.Fail(err)
	}
}
