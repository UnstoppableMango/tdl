package cli

import (
	"bytes"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/unstoppablemango/tdl/ast"
	"github.com/unstoppablemango/tdl/parser"
)

func newFmtCmd() *cobra.Command {
	var write bool

	cmd := &cobra.Command{
		Use:   "fmt <file>",
		Short: "Print a TDL file in canonical formatting",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := args[0]
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}

			file, err := parser.Parse(path, bytes.NewReader(data))
			if err != nil {
				fmt.Fprintln(cmd.ErrOrStderr(), err)
				return err
			}

			out := ast.Fprint(file)
			if write {
				return os.WriteFile(path, []byte(out), 0o644)
			}
			fmt.Fprint(cmd.OutOrStdout(), out)
			return nil
		},
	}

	cmd.Flags().BoolVarP(&write, "write", "w", false, "write result to the source file instead of stdout")
	return cmd
}
