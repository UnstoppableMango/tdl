// Command tdl-gen-go is the Go backend as a plugin.
// It serves the backend value the built-in registry holds.
package main

import (
	"fmt"
	"os"

	"github.com/unstoppablemango/tdl/backend/golang"
	"github.com/unstoppablemango/tdl/plugin"
)

func main() {
	if err := plugin.Serve(golang.Backend{}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
