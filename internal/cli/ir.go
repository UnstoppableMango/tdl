package cli

import (
	"bytes"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/unstoppablemango/tdl/internal/sema"
	"github.com/unstoppablemango/tdl/ir"
	"github.com/unstoppablemango/tdl/parser"
)

func newIrCmd() *cobra.Command {
	var format string

	cmd := &cobra.Command{
		Use:   "ir <file>",
		Short: "Print the resolved model a TDL file lowers to",
		Long: "Print the resolved model a TDL file lowers to.\n\n" +
			"The text form matches `tdl ast`, so the two read side by side.\n" +
			"The JSON form is what a plugin receives, so a backend author can\n" +
			"see the shape they will be handed.\n\n" +
			"Problems found while lowering go to stderr and set a non-zero exit\n" +
			"status, and the model is still printed: an incomplete model is\n" +
			"worth looking at when the diagnostics are what you are debugging.",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := args[0]
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}

			file, err := parser.Parse(path, bytes.NewReader(data))
			if err != nil {
				fmt.Fprintln(cmd.ErrOrStderr(), err)
				return err
			}

			model, diags := sema.Lower(file)

			out, err := renderModel(model, format)
			if err != nil {
				return err
			}
			fmt.Fprint(cmd.OutOrStdout(), out)

			if len(diags) > 0 {
				fmt.Fprintln(cmd.ErrOrStderr(), diags)
				return diags
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&format, "format", "text", "output format: text or json")
	return cmd
}

func renderModel(model *ir.Model, format string) (string, error) {
	switch format {
	case "text":
		return ir.Dump(model), nil
	case "json":
		data, err := protojson.MarshalOptions{Multiline: true, Indent: "  "}.Marshal(model)
		if err != nil {
			return "", err
		}
		return string(data) + "\n", nil
	default:
		return "", fmt.Errorf("unknown format %q: want text or json", format)
	}
}
