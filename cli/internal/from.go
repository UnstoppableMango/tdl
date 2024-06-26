package cli

import (
	"io"
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
		Use:   "from [file...]",
		Short: "Generate a UMl spec from source code",
		Long:  `Searches file(s) for types and generates a UMl spec describing them`,
		RunE: func(cmd *cobra.Command, args []string) error {
			log := GetLogger(cmd)
			opts := FromCmdOptions{Log: log}
			ctx := cmd.Context()

			log.Debug("creating converter")
			conv, err := create(ctx, opts, args)
			if err != nil {
				return err
			}

			var input io.Reader = os.Stdin
			if len(args) > 1 {
				log.Debug("found file arguments")
				// TODO: Accept more files
				input, err = os.Open(args[1])
				if err != nil {
					return err
				}
			}

			log.Debug("executing converter")
			spec, err := conv.From(ctx, input)
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
