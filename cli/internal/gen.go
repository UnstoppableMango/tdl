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
		Use: "gen",
		RunE: func(cmd *cobra.Command, args []string) error {
			log := GetLogger(cmd)
			opts := GenCmdOptions{Log: log}
			ctx := cmd.Context()

			log.Debug("reading stdin")
			data, err := io.ReadAll(os.Stdin)
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
