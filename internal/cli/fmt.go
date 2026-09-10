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
			"a CI job or a pre-commit hook wants.\n\n" +
			"A file named - is read from standard input, which -w rejects:\n" +
			"there is nothing to write it back to.",
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if write {
				for _, path := range args {
					if isStdin(path) {
						return fmt.Errorf("-w has nothing to write back to when reading %s", stdinName)
					}
				}
			}

			staleFiles, staleStdin := 0, false
			header := newHeader(args)
			err := eachFile(cmd, args, func(path string) error {
				src, file, err := readFile(cmd, path)
				if err != nil {
					return err
				}
				out := ast.Fprint(file)

				switch {
				case check:
					if out != src {
						if isStdin(path) {
							staleStdin = true
						} else {
							staleFiles++
						}
						fmt.Fprintln(cmd.OutOrStdout(), displayName(path))
					}
					return nil

				case write:
					return writeFormatted(path, out)

				default:
					header.write(cmd, path)
					fmt.Fprint(cmd.OutOrStdout(), out)
					return nil
				}
			})
			if err != nil {
				return err
			}

			if staleFiles > 0 || staleStdin {
				return staleError(staleFiles, staleStdin)
			}
			return nil
		},
	}

	cmd.Flags().BoolVarP(&write, "write", "w", false, "write result to the source file instead of stdout")
	cmd.Flags().BoolVarP(&check, "check", "l", false, "list files that are not canonically formatted, writing nothing")
	cmd.MarkFlagsMutuallyExclusive("write", "check")
	return cmd
}

// staleError describes what --check found. -w has nothing to write standard
// input back to, so the hint only names it when a file on disk is stale.
func staleError(files int, stdin bool) error {
	switch {
	case files == 0:
		return fmt.Errorf("%s is not formatted", stdinName)
	case stdin:
		return fmt.Errorf("%s and %d file(s) are not formatted, run \"tdl fmt -w <file>...\" to fix the files", stdinName, files)
	default:
		return fmt.Errorf("%d file(s) are not formatted, run \"tdl fmt -w <file>...\" to fix them", files)
	}
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
