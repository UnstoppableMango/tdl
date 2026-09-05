package cli

import (
	"bytes"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/unstoppablemango/tdl/lex"
)

func newTokensCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "tokens <file>...",
		Short: "Print the token stream the lexer produces for a TDL file",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return eachFile(cmd, args, func(i int, path string) error {
				// The lexer is the stage under test here, so this reads the
				// file itself rather than going through loadFile, which would
				// parse it and fail on a file whose tokens are worth seeing.
				data, err := os.ReadFile(path)
				if err != nil {
					return err
				}

				writeHeader(cmd, args, i, path)
				fmt.Fprint(cmd.OutOrStdout(), dumpTokens(path, string(data)))
				return nil
			})
		},
	}
}

// dumpTokens lexes src to completion and renders one aligned row per
// token: position, kind, and the literal text the lexer captured.
func dumpTokens(filename, src string) string {
	var b bytes.Buffer
	w := tabwriter.NewWriter(&b, 0, 0, 2, ' ', 0)

	lx := lex.New(filename, src)
	for {
		tok := lx.Next()
		text := tok.Text
		if text != "" {
			text = fmt.Sprintf("%q", text)
		}
		fmt.Fprintf(w, "%d:%d\t%s\t%s\n", tok.Pos.Line, tok.Pos.Col, tok.Kind, text)
		if tok.Kind == lex.EOF {
			break
		}
	}

	_ = w.Flush()
	return b.String()
}
