package cli

import (
	"io"
	"log/slog"
	"os"

	"github.com/spf13/cobra"
	"github.com/unstoppablemango/tdl/pkg/uml"
	"google.golang.org/protobuf/proto"
)

type GenCmdOptions struct {
	uml.GeneratorOptions
	Log *slog.Logger
}

func NewGenCmd[T uml.NewGenerator[GenCmdOptions]](create T) *cobra.Command {
	return &cobra.Command{
		Use:   "gen [spec...]",
		Short: "Generate source code types from the supplied spec(s)",
		Long:  `Generate source code types from the supplied spec(s)`,
		RunE: func(cmd *cobra.Command, args []string) error {
			log := GetLogger(cmd)
			opts := GenCmdOptions{Log: log}
			ctx := cmd.Context()

			var err error
			var input io.Reader = os.Stdin
			if len(args) > 1 {
				log.Debug("found file arguments")
				// TODO: Accept more files
				input, err = os.Open(args[1])
				if err != nil {
					return err
				}
			}

			log.Debug("reading input")
			data, err := io.ReadAll(input)
			if err != nil {
				return err
			}

			log.Debug("unmarhshalling input")
			spec := uml.Spec{}
			if err = proto.Unmarshal(data, &spec); err != nil {
				return err
			}

			log.Debug("creating generator")
			gen, err := create(ctx, opts, args)
			if err != nil {
				return err
			}

			log.Debug("executing generator")
			return gen.Gen(ctx, &spec, os.Stdout)
		},
	}
}
