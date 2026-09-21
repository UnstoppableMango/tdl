// Command tdl-gen-salesforce is the Salesforce backend as a plugin.
//
// It is the same backend value the built-in registry holds, served over a
// connection instead of called in process.
package main

import (
	"fmt"
	"os"

	"github.com/unstoppablemango/tdl/backend/salesforce"
	"github.com/unstoppablemango/tdl/plugin"
)

func main() {
	if err := plugin.Serve(salesforce.Backend{}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
