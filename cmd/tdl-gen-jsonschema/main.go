// Command tdl-gen-jsonschema is the JSON Schema backend as a plugin.
// It serves the backend value the built-in registry holds.
package main

import (
	"fmt"
	"os"

	"github.com/unstoppablemango/tdl/backend/jsonschema"
	"github.com/unstoppablemango/tdl/plugin"
)

func main() {
	if err := plugin.Serve(jsonschema.Backend{}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
