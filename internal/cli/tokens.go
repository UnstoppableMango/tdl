package cli

import (
	"bytes"
	"fmt"
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
			header := newHeader(args)
			return eachFile(cmd, args, func(path string) error {
				// The lexer is the stage under test here, so this stops
				// short of loadFile, which would parse the file and fail on
				// one whose tokens are worth seeing.
				data, err := readSource(cmd, path)
				if err != nil {
					return err
				}

				header.write(cmd, path)
				fmt.Fprint(cmd.OutOrStdout(), dumpTokens(displayName(path), string(data)))
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
