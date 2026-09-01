package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

// toolVersion is written by release-please. The annotation is what tells
// it which string to replace, so this is the one place the tool's version
// is stated and nothing has to be kept in step by hand.
//
// specVersion is not written by release-please and must not be: it tracks
// docs/spec.md, which moves on its own schedule. A release that changes no
// spec text should not claim to have changed the spec.
const (
	toolVersion = "0.1.2" // x-release-please-version
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
