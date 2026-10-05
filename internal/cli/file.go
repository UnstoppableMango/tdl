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

// stdinArg is the path that names standard input.
const stdinArg = "-"

// stdinName names standard input in positions and errors, since `-` reads
// as a flag.
const stdinName = "<stdin>"

func isStdin(path string) bool { return path == stdinArg }

func displayName(path string) string {
	if isStdin(path) {
		return stdinName
	}
	return path
}

// loadFile reads and parses one source file. Every command that works on a
// parse tree goes through it.
func loadFile(cmd *cobra.Command, path string) (*ast.File, error) {
	_, file, err := readFile(cmd, path)
	return file, err
}

// readFile is loadFile that also returns the source text.
func readFile(cmd *cobra.Command, path string) (string, *ast.File, error) {
	data, err := readSource(cmd, path)
	if err != nil {
		return "", nil, err
	}

	file, err := parser.Parse(displayName(path), bytes.NewReader(data))
	return string(data), file, err
}

// readSource reads path, or standard input for `-`, without parsing.
func readSource(cmd *cobra.Command, path string) ([]byte, error) {
	if isStdin(path) {
		return io.ReadAll(cmd.InOrStdin())
	}
	return os.ReadFile(path)
}

// eachFile runs fn over every path, printing each failure and continuing.
// The returned error only counts failures, since each printed error already
// names its file.
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

// header writes a `==> path <==` banner before each file's output, as
// head(1) does, and nothing for a single file. The blank line goes before
// every banner after the first one written, so a file that failed before
// printing leaves no gap.
type header struct {
	several bool
	written bool
}

func newHeader(paths []string) *header {
	return &header{several: len(paths) > 1}
}

func (h *header) write(cmd *cobra.Command, path string) {
	if !h.several {
		return
	}
	if h.written {
		fmt.Fprintln(cmd.OutOrStdout())
	}
	h.written = true
	fmt.Fprintf(cmd.OutOrStdout(), "==> %s <==\n", displayName(path))
}
