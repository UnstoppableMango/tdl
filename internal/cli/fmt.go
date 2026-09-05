package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/unstoppablemango/tdl/ast"
)

func newFmtCmd() *cobra.Command {
	var write bool

	cmd := &cobra.Command{
		Use:   "fmt <file>...",
		Short: "Print a TDL file in canonical formatting",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return eachFile(cmd, args, func(i int, path string) error {
				file, err := loadFile(path)
				if err != nil {
					return err
				}

				out := ast.Fprint(file)
				if write {
					return os.WriteFile(path, []byte(out), 0o644)
				}

				writeHeader(cmd, args, i, path)
				fmt.Fprint(cmd.OutOrStdout(), out)
				return nil
			})
		},
	}

	cmd.Flags().BoolVarP(&write, "write", "w", false, "write result to the source file instead of stdout")
	return cmd
}
