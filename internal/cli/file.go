package cli

import (
	"bytes"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/unstoppablemango/tdl/ast"
	"github.com/unstoppablemango/tdl/parser"
)

// loadFile reads and parses one source file. Every command that works on a
// parse tree starts here, so they agree on how a file is read and on what a
// parse error looks like.
func loadFile(path string) (*ast.File, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return parser.Parse(path, bytes.NewReader(data))
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
	fmt.Fprintf(cmd.OutOrStdout(), "==> %s <==\n", path)
}
