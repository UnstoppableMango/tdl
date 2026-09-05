package cli

import (
	"bytes"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/unstoppablemango/tdl/ast"
	"github.com/unstoppablemango/tdl/parser"
)

// stdinArg is the path that names standard input, the convention every
// tool that reads a file list uses.
const stdinArg = "-"

// stdinName is what standard input is called in a position and in an error,
// since `-` reads as a flag and as a file that is not there.
const stdinName = "<stdin>"

// isStdin reports whether path names standard input.
func isStdin(path string) bool { return path == stdinArg }

// displayName is what a path is called in output.
func displayName(path string) string {
	if isStdin(path) {
		return stdinName
	}
	return path
}

// loadFile reads and parses one source file. Every command that works on a
// parse tree starts here, so they agree on how a file is read, on what
// reading `-` means, and on what a parse error looks like.
func loadFile(cmd *cobra.Command, path string) (*ast.File, error) {
	_, file, err := readFile(cmd, path)
	return file, err
}

// readFile is loadFile with the source text it read, for a caller that has
// to compare against what was on disk rather than only against the tree.
func readFile(cmd *cobra.Command, path string) (string, *ast.File, error) {
	var (
		data []byte
		err  error
	)
	if isStdin(path) {
		data, err = io.ReadAll(cmd.InOrStdin())
	} else {
		data, err = os.ReadFile(path)
	}
	if err != nil {
		return "", nil, err
	}

	file, err := parser.Parse(displayName(path), bytes.NewReader(data))
	return string(data), file, err
}

// eachFile runs fn over every path given.
//
// A failing file is reported and the walk continues, the way the parser
// reports every syntax error in a file rather than the first: a run over
// twenty files says which of them are broken, not which one is broken
// first. The error returned counts them rather than repeating them, since
// a diagnostic and an os.PathError both already name the file they are
// about.
func eachFile(cmd *cobra.Command, paths []string, fn func(i int, path string) error) error {
	failed := 0
	for i, path := range paths {
		if err := fn(i, path); err != nil {
			fmt.Fprintln(cmd.ErrOrStderr(), err)
			failed++
		}
	}

	switch failed {
	case 0:
		return nil
	case 1:
		return fmt.Errorf("1 file failed")
	default:
		return fmt.Errorf("%d files failed", failed)
	}
}

// writeHeader prints a `==> path <==` banner, the way head(1) separates the
// files it was given. It writes nothing for a single file, so output stays
// pipeable in the common case.
func writeHeader(cmd *cobra.Command, paths []string, i int, path string) {
	if len(paths) < 2 {
		return
	}
	if i > 0 {
		fmt.Fprintln(cmd.OutOrStdout())
	}
	fmt.Fprintf(cmd.OutOrStdout(), "==> %s <==\n", displayName(path))
}
