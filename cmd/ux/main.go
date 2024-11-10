package main

import (
	"fmt"
	"os"

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
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}
