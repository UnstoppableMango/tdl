package cli

import (
	"io"
	"os"

	"github.com/spf13/cobra"
	"github.com/unstoppablemango/tdl/pkg/uml"
	"google.golang.org/protobuf/proto"
)

func NewFromCmd[T uml.NewConverter[uml.ConverterOptions]](create T) *cobra.Command {
	return &cobra.Command{
		Use: "from",
		RunE: func(cmd *cobra.Command, args []string) error {
			opts := uml.ConverterOptions{}
			ctx := cmd.Context()
			conv, err := create(ctx, opts, args)
			if err != nil {
				return err
			}

			spec, err := conv.From(cmd.Context(), os.Stdin)
			if err != nil {
				return err
			}

			data, err := proto.Marshal(spec)
			if err != nil {
				return err
			}

			if _, err = os.Stdout.Write(data); err != nil {
				return err
			}

			return nil
		},
	}
}

func NewGenCmd[T uml.NewGenerator[uml.GeneratorOptions]](create T) *cobra.Command {
	return &cobra.Command{
		Use: "gen",
		RunE: func(cmd *cobra.Command, args []string) error {
			opts := uml.GeneratorOptions{}
			data, err := io.ReadAll(os.Stdin)
			if err != nil {
				return err
			}

			spec := uml.Spec{}
			if err = proto.Unmarshal(data, &spec); err != nil {
				return err
			}

			ctx := cmd.Context()
			gen, err := create(ctx, opts, args)
			if err != nil {
				return err
			}

			return gen.Gen(ctx, &spec, os.Stdout)
		},
	}
}
