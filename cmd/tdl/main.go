// Command tdl parses and formats Type Description Language source files.
package main

import (
	"fmt"
	"os"

	"github.com/unstoppablemango/tdl/internal/cli"
)

func main() {
	// The root command silences cobra's error printing, so every error is
	// printed here, once.
	if err := cli.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
