// Command tdl-gen-debug is the debug backend as a plugin.
//
// It is the same backend value the built-in registry holds, served over a
// connection instead of called in process. That is what makes "one
// protocol, two hosts" testable: the two paths differ in transport and in
// nothing else.
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
