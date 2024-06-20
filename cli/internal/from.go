package cli

import (
	"log/slog"
	"os"

	"github.com/spf13/cobra"
	"github.com/unstoppablemango/tdl/pkg/uml"
	"google.golang.org/protobuf/proto"
)

type FromCmdOptions struct {
	uml.ConverterOptions
	Log *slog.Logger
}

func NewFromCmd[T uml.NewConverter[FromCmdOptions]](create T) *cobra.Command {
	return &cobra.Command{
		Use: "from",
		RunE: func(cmd *cobra.Command, args []string) error {
			log := GetLogger(cmd)
			opts := FromCmdOptions{Log: log}
			ctx := cmd.Context()

			log.Debug("creating converter")
			conv, err := create(ctx, opts, args)
			if err != nil {
				return err
			}

			log.Debug("executing converter")
			spec, err := conv.From(ctx, os.Stdin)
			if err != nil {
				return err
			}

			log.Debug("marshalling result")
			data, err := proto.Marshal(spec)
			if err != nil {
				return err
			}

			log.Debug("writing result")
			if _, err = os.Stdout.Write(data); err != nil {
				return err
			}

			return nil
		},
	}
}
