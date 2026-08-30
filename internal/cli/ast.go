package cli

import (
	"bytes"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/unstoppablemango/tdl/ast"
	"github.com/unstoppablemango/tdl/parser"
)

func newAstCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "ast <file>",
		Short: "Print the parse tree a TDL file produces",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := args[0]
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}

			file, err := parser.Parse(path, bytes.NewReader(data))
			if err != nil {
				return err
			}

			fmt.Fprint(cmd.OutOrStdout(), ast.Dump(file))
			return nil
		},
	}
}
