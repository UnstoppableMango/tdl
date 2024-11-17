package testing

import (
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"github.com/unstoppablemango/tdl/internal/util"
	"github.com/unstoppablemango/tdl/pkg/conform"
)

func NewConform() *cobra.Command {
	return &cobra.Command{
		Use:   "conform [TARGET]",
		Short: "Run conformance tests against TARGET",
		Long: `This command will attempt to identify the type of
endpoint referred to by TARGET and perform the conformance
tests against it. Currently only supports a path to a binary
that communicates via stdin/stdout`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			err := conform.Execute(afero.NewOsFs(), args[0], args[1:])
			if err != nil {
				util.Fail(err)
			}
		},
	}
}
