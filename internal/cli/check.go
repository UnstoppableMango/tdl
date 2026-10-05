package cli

import (
	"github.com/spf13/cobra"

	"github.com/unstoppablemango/tdl/internal/sema"
)

func newCheckCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "check <file>...",
		Short: "Parse and lower a TDL file and report every problem found",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return eachFile(cmd, args, func(path string) error {
				file, err := loadFile(cmd, path)
				if err != nil {
					return err
				}
				if _, diags := sema.Lower(file, sema.WithLoader(sema.FSLoader{})); len(diags) > 0 {
					return diags
				}
				return nil
			})
		},
	}
}
