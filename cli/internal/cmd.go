package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/unstoppablemango/tdl/pkg/uml"
)

func NewFromCmd(create uml.NewConverter) *cobra.Command {
	return &cobra.Command{
		Use:  "from",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts := uml.ConverterOptions{}
			uml.Apply(&opts, uml.WithMimeType(args[0]))
			conv := create(opts)
			spec, err := conv.From(cmd.Context(), os.Stdin)
			fmt.Printf("output:\n%s\n", spec)
			return err
		},
	}
}

func NewGenCmd(create uml.NewGenerator) *cobra.Command {
	return &cobra.Command{
		Use:  "gen",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts := uml.GeneratorOptions{}
			uml.Apply(&opts, uml.WithTarget(args[0]))
			gen := create(opts)
			return gen.Gen(cmd.Context(), &uml.Spec{}, os.Stdout)
		},
	}
}
