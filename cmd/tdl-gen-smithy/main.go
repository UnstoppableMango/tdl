// Command tdl-gen-smithy is the Smithy backend as a plugin.
//
// It is the same backend value the built-in registry holds, served over a
// connection instead of called in process.
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
