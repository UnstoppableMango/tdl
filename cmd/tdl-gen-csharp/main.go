// Command tdl-gen-csharp is the C# backend as a plugin.
// It serves the backend value the built-in registry holds.
package main

import (
	"fmt"
	"os"

	"github.com/unstoppablemango/tdl/backend/csharp"
	"github.com/unstoppablemango/tdl/plugin"
)

func main() {
	if err := plugin.Serve(csharp.Backend{}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
