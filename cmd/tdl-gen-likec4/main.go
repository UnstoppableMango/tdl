// Command tdl-gen-likec4 is the LikeC4 backend as a plugin.
// It serves the backend value the built-in registry holds.
package main

import (
	"fmt"
	"os"

	"github.com/unstoppablemango/tdl/backend/likec4"
	"github.com/unstoppablemango/tdl/plugin"
)

func main() {
	if err := plugin.Serve(likec4.Backend{}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
