// Command tdl-gen-thrift is the Thrift backend as a plugin.
//
// It is the same backend value the built-in registry holds, served over a
// connection instead of called in process.
package main

import (
	"fmt"
	"os"

	"github.com/unstoppablemango/tdl/backend/thrift"
	"github.com/unstoppablemango/tdl/plugin"
)

func main() {
	if err := plugin.Serve(thrift.Backend{}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
