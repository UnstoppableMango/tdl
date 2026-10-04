// Command tdl-gen-smithy is the Smithy backend as a plugin.
// It serves the backend value the built-in registry holds.
package main

import (
	"fmt"
	"os"

	"github.com/unstoppablemango/tdl/backend/smithy"
	"github.com/unstoppablemango/tdl/plugin"
)

func main() {
	if err := plugin.Serve(smithy.Backend{}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
