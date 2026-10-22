// Command tdl-gen-typescript is the TypeScript backend as a plugin.
// It serves the backend value the built-in registry holds.
//
// Run as `tdl-gen-typescript content-mapper`, it is instead the
// TypeScript 7.1 content mapper that lets tsc import a .tdl file.
package main

import (
	"fmt"
	"os"

	"github.com/unstoppablemango/tdl/backend/typescript"
	"github.com/unstoppablemango/tdl/plugin"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	switch {
	case len(args) == 0:
		return plugin.Serve(typescript.Backend{})
	case len(args) == 1 && args[0] == "content-mapper":
		return typescript.Mapper{}.Serve(os.Stdin, os.Stdout)
	}
	return fmt.Errorf("usage: tdl-gen-typescript [content-mapper]")
}
