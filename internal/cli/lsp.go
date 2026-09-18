package cli

import (
	"io"

	"github.com/spf13/cobra"

	"github.com/unstoppablemango/tdl/internal/lsp"
)

func newLspCmd() *cobra.Command {
	var preludePath string

	cmd := &cobra.Command{
		Use:   "lsp",
		Short: "Serve the Language Server Protocol over stdin and stdout",
		Long: "Serve the Language Server Protocol over stdin and stdout.\n\n" +
			"An editor starts this and speaks JSON-RPC to it. It is not meant\n" +
			"to be run by hand: with a terminal on both sides it sits waiting\n" +
			"for a request that never comes.\n\n" +
			"It reports what `tdl check` reports, against the text in the\n" +
			"editor rather than the text on disk.\n\n" +
			"--prelude lowers against a different prelude, the way it does\n" +
			"everywhere else.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			opts, err := preludeOptions(preludePath)
			if err != nil {
				return err
			}
			return lsp.Serve(cmd.Context(), stdio{cmd.InOrStdin(), cmd.OutOrStdout()}, opts...)
		},
	}

	cmd.Flags().StringVar(&preludePath, "prelude", "", "lower against this prelude instead of the built-in one")
	return cmd
}

// stdio is the connection an editor speaks over: this process's input and
// output as one stream.
type stdio struct {
	in  io.Reader
	out io.Writer
}

func (s stdio) Read(p []byte) (int, error)  { return s.in.Read(p) }
func (s stdio) Write(p []byte) (int, error) { return s.out.Write(p) }

// Close closes nothing. The streams belong to the process, which exits
// when the connection ends.
func (stdio) Close() error { return nil }
