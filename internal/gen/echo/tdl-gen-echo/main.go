// Command tdl-gen-echo is the echo test backend as a plugin.
package main

import (
	"fmt"
	"os"

	"github.com/unstoppablemango/tdl/internal/gen/echo"
	"github.com/unstoppablemango/tdl/plugin"
)

func main() {
	if err := plugin.Serve(echo.Backend{}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
