// Command tdl-gen-typescript is the TypeScript backend as a plugin.
// It serves the backend value the built-in registry holds.
package main

import (
	"fmt"
	"os"

	"github.com/unstoppablemango/tdl/backend/typescript"
	"github.com/unstoppablemango/tdl/plugin"
)

func main() {
	if err := plugin.Serve(typescript.Backend{}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
