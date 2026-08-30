// Command tdl parses and formats Type Description Language source files.
package main

import (
	"fmt"
	"os"

	"github.com/unstoppablemango/tdl/internal/cli"
)

func main() {
	// The root command silences cobra's own error printing so that a
	// diagnostic list renders as itself rather than wrapped in "Error:".
	// Printing here means an error from anywhere is reported once, instead
	// of only the ones a command remembered to print for itself.
	if err := cli.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
