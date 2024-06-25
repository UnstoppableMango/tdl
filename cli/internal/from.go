package cli

import (
	"io"
	"os"

	"github.com/spf13/cobra"
	"github.com/unstoppablemango/tdl/pkg/uml"
)

func NewFromCmd(create func(uml.ConverterOptions) (uml.Converter, error)) *cobra.Command {
	return &cobra.Command{
		Use:   "from [file...]",
		Short: "Generate a UMl spec from source code",
		Long:  `Searches file(s) for types and generates a UMl spec describing them`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			exec := &runnerCmd{
				args: args,
				log:  GetLogger(cmd),
			}

			return exec.run(func(key string, input io.Reader) error {
				log := exec.log.With("key", key)

				log.Debug("guessing media type")
				mediaType, err := uml.GuessMediaType(key)
				if err != nil {
					return err
				}

				log.Debug("creating converter")
				converter, err := create(uml.ConverterOptions{
					MediaType: &mediaType,
					Log:       log,
				})
				if err != nil {
					return err
				}

				log.Debug("executing converter")
				spec, err := converter.From(ctx, input)
				if err != nil {
					return err
				}

				log.Debug("marshalling result")
				data, err := uml.Marshal(mediaType, spec)
				if err != nil {
					return err
				}

				log.Debug("writing result to stdout")
				_, err = os.Stdout.Write(data)

				return err
			})
		},
	}
}
