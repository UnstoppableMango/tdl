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
	_, file, err := readFile(path)
	return file, err
}

// readFile is loadFile with the source text it read, for a caller that has
// to compare against what was on disk rather than only against the tree.
func readFile(path string) (string, *ast.File, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", nil, err
	}
	file, err := parser.Parse(path, bytes.NewReader(data))
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
func eachFile(cmd *cobra.Command, paths []string, fn func(path string) error) error {
	failed := 0
	for _, path := range paths {
		if err := fn(path); err != nil {
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

// header separates the output of several files with a `==> path <==`
// banner, the way head(1) separates the files it was given.
//
// It writes nothing for a single file, so output stays pipeable in the
// common case. A blank line goes before every banner after the first one
// written, rather than before every file after the first one given: a file
// that fails before printing anything leaves no gap.
type header struct {
	several bool
	written bool
}

// newHeader returns a header for the files a command was given.
func newHeader(paths []string) *header {
	return &header{several: len(paths) > 1}
}

// write prints the banner for path.
func (h *header) write(cmd *cobra.Command, path string) {
	if !h.several {
		return
	}
	if h.written {
		fmt.Fprintln(cmd.OutOrStdout())
	}
	h.written = true
	fmt.Fprintf(cmd.OutOrStdout(), "==> %s <==\n", path)
}
