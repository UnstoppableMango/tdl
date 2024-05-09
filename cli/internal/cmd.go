package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/unstoppablemango/tdl/pkg/uml"
)

func NewFromCmd(conv uml.Converter) *cobra.Command {
	return &cobra.Command{
		Use:  "from",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			spec, err := conv.From(ctx, os.Stdin)
			fmt.Printf("output:\n%s\n", spec)
			return err
		},
	}
}

func NewGenCmd(gen uml.Generator) *cobra.Command {
	return &cobra.Command{
		Use:  "gen",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			err := gen.Gen(ctx, &uml.Spec{}, os.Stdout)
			return err
		},
	}
}
