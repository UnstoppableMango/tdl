package main

import (
	"fmt"
	"os"

	"github.com/unstoppablemango/tdl/pkg/cmd"
	"github.com/unstoppablemango/tdl/pkg/gen/io"
)

func main() {
	rootCmd := cmd.NewUx()
	rootCmd.AddCommand(
		cmd.NewConform(),
		cmd.NewGenFor(io.BinFromPath),
	)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}
