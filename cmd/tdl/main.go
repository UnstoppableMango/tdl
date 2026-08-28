// Command tdl parses and formats Type Description Language source files.
package main

import (
	"os"

	"github.com/unstoppablemango/tdl/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}
