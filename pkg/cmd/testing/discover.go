package testing

import (
	"fmt"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"github.com/unstoppablemango/tdl/internal/util"
	"github.com/unstoppablemango/tdl/pkg/testing"
)

func NewDiscover() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "discover [PATH]",
		Short: "Search for tests at PATH",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			tests, err := testing.Discover(
				afero.NewOsFs(),
				args[0],
			)
			if err != nil {
				util.Fail(err)
			}

			for _, t := range tests {
				fmt.Println(t.Name)
			}
		},
	}

	return cmd
}
