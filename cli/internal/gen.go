package cli

import (
	"io"
	"os"

	"github.com/spf13/cobra"
	"github.com/unstoppablemango/tdl/pkg/uml"
)

func NewGenCmd(generator uml.Generator) *cobra.Command {
	return &cobra.Command{
		Use:   "gen [spec...]",
		Short: "Generate source code types from the supplied spec(s)",
		Long:  `Generate source code types from the supplied spec(s)`,
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

				log.Debug("reading input")
				data, err := io.ReadAll(input)
				if err != nil {
					return err
				}

				spec := &uml.Spec{}
				log.Debug("unmarshalling spec", "mediaType", mediaType)
				if err = uml.Unmarshal(mediaType, data, spec); err != nil {
					return err
				}

				log.Debug("executing generator")
				return generator.Gen(ctx, spec, os.Stdout)
			})
		},
	}
}
