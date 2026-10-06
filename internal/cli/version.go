package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

// toolVersion is written by release-please through its annotation.
//
// specVersion tracks docs/spec.md and must not be given the annotation: a
// release that changes no spec text has not changed the spec.
const (
	toolVersion = "0.3.1" // x-release-please-version
	specVersion = "0.1.0-draft"
)

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the tdl tool and spec versions",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprintf(cmd.OutOrStdout(), "tdl %s (spec %s)\n", toolVersion, specVersion)
			return nil
		},
	}
}
