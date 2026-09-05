package cli

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/unstoppablemango/tdl/ast"
)

func newFmtCmd() *cobra.Command {
	var (
		write bool
		check bool
	)

	cmd := &cobra.Command{
		Use:   "fmt <file>...",
		Short: "Print a TDL file in canonical formatting",
		Long: "Print a TDL file in canonical formatting.\n\n" +
			"-w writes the result back, keeping the file's mode.\n\n" +
			"--check writes nothing and lists the files that are not already\n" +
			"canonical, exiting non-zero when it lists any. That is the form\n" +
			"a CI job or a pre-commit hook wants.",
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			stale := 0
			err := eachFile(cmd, args, func(i int, path string) error {
				src, file, err := readFile(path)
				if err != nil {
					return err
				}
				out := ast.Fprint(file)

				switch {
				case check:
					if out != src {
						stale++
						fmt.Fprintln(cmd.OutOrStdout(), path)
					}
					return nil

				case write:
					return writeFormatted(path, out)

				default:
					writeHeader(cmd, args, i, path)
					fmt.Fprint(cmd.OutOrStdout(), out)
					return nil
				}
			})
			if err != nil {
				return err
			}

			if stale > 0 {
				return fmt.Errorf("%d file(s) are not formatted; run: tdl fmt -w", stale)
			}
			return nil
		},
	}

	cmd.Flags().BoolVarP(&write, "write", "w", false, "write result to the source file instead of stdout")
	cmd.Flags().BoolVarP(&check, "check", "l", false, "list files that are not canonically formatted, writing nothing")
	cmd.MarkFlagsMutuallyExclusive("write", "check")
	return cmd
}

// writeFormatted replaces path with formatted, keeping the mode the file
// already had. Formatting is not the place to widen a file's permissions.
func writeFormatted(path, formatted string) error {
	mode := os.FileMode(0o644)
	switch info, err := os.Stat(path); {
	case err == nil:
		mode = info.Mode().Perm()
	case !errors.Is(err, os.ErrNotExist):
		return err
	}
	return os.WriteFile(path, []byte(formatted), mode)
}
