package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

// toolVersion and specVersion are hardcoded for the M1 draft. A later
// milestone can wire toolVersion to a build-time ldflag.
const (
	toolVersion = "0.1.0-dev"
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
