package cli

import (
	"io"
	"os"

	"github.com/spf13/cobra"
	"github.com/unstoppablemango/tdl/pkg/uml"
	"google.golang.org/protobuf/proto"
)

func NewFromCmd(create uml.NewConverter) *cobra.Command {
	return &cobra.Command{
		Use: "from",
		RunE: func(cmd *cobra.Command, args []string) error {
			opts := uml.ConverterOptions{}
			spec, err := create(opts).From(cmd.Context(), os.Stdin)
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

func NewGenCmd(create uml.NewGenerator) *cobra.Command {
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

			return create(opts).Gen(cmd.Context(), &spec, os.Stdout)
		},
	}
}
