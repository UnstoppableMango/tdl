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
		Use:   "tokens <file>",
		Short: "Print the token stream the lexer produces for a TDL file",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := args[0]
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}

			fmt.Fprint(cmd.OutOrStdout(), dumpTokens(path, string(data)))
			return nil
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
