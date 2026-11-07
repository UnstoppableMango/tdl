// Command tdl-gen-yolo is the delegated backend as a plugin. It is never
// compiled into tdl, so tdl links no model client.
package main

import (
	"fmt"
	"os"

	"github.com/unstoppablemango/tdl/backend/yolo"
	"github.com/unstoppablemango/tdl/plugin"
)

func main() {
	if err := plugin.Serve(yolo.Backend{}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
