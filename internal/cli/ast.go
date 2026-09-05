package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/unstoppablemango/tdl/ast"
)

func newAstCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "ast <file>...",
		Short: "Print the parse tree a TDL file produces",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return eachFile(cmd, args, func(i int, path string) error {
				file, err := loadFile(cmd, path)
				if err != nil {
					return err
				}

				writeHeader(cmd, args, i, path)
				fmt.Fprint(cmd.OutOrStdout(), ast.Dump(file))
				return nil
			})
		},
	}
}
