// Command tdl-gen-nix is the Nix backend as a plugin.
// It serves the backend value the built-in registry holds.
package main

import (
	"fmt"
	"os"

	"github.com/unstoppablemango/tdl/backend/nix"
	"github.com/unstoppablemango/tdl/plugin"
)

func main() {
	if err := plugin.Serve(nix.Backend{}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
