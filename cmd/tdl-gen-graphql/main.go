// Command tdl-gen-graphql is the GraphQL backend as a plugin.
// It serves the backend value the built-in registry holds.
package main

import (
	"fmt"
	"os"

	"github.com/unstoppablemango/tdl/backend/graphql"
	"github.com/unstoppablemango/tdl/plugin"
)

func main() {
	if err := plugin.Serve(graphql.Backend{}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
