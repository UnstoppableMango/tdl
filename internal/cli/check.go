package cli

import (
	"github.com/spf13/cobra"
)

func newCheckCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "check <file>...",
		Short: "Parse a TDL file and report every syntax error found",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return eachFile(cmd, args, func(_ int, path string) error {
				_, err := loadFile(cmd, path)
				return err
			})
		},
	}
}
