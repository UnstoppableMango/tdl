package testing

import (
	"fmt"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"github.com/unstoppablemango/tdl/internal/util"
	"github.com/unstoppablemango/tdl/pkg/conform"
	"github.com/unstoppablemango/tdl/pkg/logging"
)

func NewList() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List available test suites",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			logging.Init()
			ctx := cmd.Context()
			path, err := conform.LocalGitPath(ctx)
			if err != nil {
				util.Fail(err)
			}

			fs := afero.NewOsFs()
			suites, err := afero.ReadDir(fs, path)
			if err != nil {
				util.Fail(err)
			}

			for _, info := range suites {
				suite, err := conform.ReadLocalGitSuite(ctx, fs, info.Name())
				if err != nil {
					util.Fail(err)
				}

				var count int
				for _ = range suite.Tests() {
					count++
				}

				fmt.Printf("%s: %d tests\n", suite.Name(), count)
			}
		},
	}
}
