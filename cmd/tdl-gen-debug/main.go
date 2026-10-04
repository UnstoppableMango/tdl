// Command tdl-gen-debug is the debug backend as a plugin.
// It serves the backend value the built-in registry holds.
package main

import (
	"fmt"
	"os"

	"github.com/unstoppablemango/tdl/backend/debug"
	"github.com/unstoppablemango/tdl/plugin"
)

func main() {
	if err := plugin.Serve(debug.Backend{}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
