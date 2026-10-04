// Command tdl-gen-protobuf is the protobuf backend as a plugin.
// It serves the backend value the built-in registry holds.
package main

import (
	"fmt"
	"os"

	"github.com/unstoppablemango/tdl/backend/protobuf"
	"github.com/unstoppablemango/tdl/plugin"
)

func main() {
	if err := plugin.Serve(protobuf.Backend{}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
