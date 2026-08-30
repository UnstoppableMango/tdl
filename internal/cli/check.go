package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/unstoppablemango/tdl/parser"
)

func newCheckCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "check <file>",
		Short: "Parse a TDL file and report every syntax error found",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := args[0]
			f, err := os.Open(path)
			if err != nil {
				return err
			}
			defer f.Close()

			if _, err := parser.Parse(path, f); err != nil {
				fmt.Fprintln(cmd.ErrOrStderr(), err)
				return err
			}
			return nil
		},
	}
}
