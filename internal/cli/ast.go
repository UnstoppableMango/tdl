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
			header := newHeader(args)
			return eachFile(cmd, args, func(path string) error {
				file, err := loadFile(cmd, path)
				if err != nil {
					return err
				}

				header.write(cmd, path)
				fmt.Fprint(cmd.OutOrStdout(), ast.Dump(file))
				return nil
			})
		},
	}
}
