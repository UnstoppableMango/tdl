package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/unstoppablemango/tdl/internal"
	"github.com/unstoppablemango/tdl/internal/util"
	"github.com/unstoppablemango/tdl/pkg/plugin"
	"github.com/unstoppablemango/tdl/pkg/target"
)

func NewWhich() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "which [TARGET]",
		Short: "Print the name of the closest matching plugin",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			internal.InitLogging()
			t, err := target.Parse(args[0])
			if err != nil {
				util.Fail(err)
			}

			g, err := t.Choose(plugin.Static())
			if err != nil {
				util.Fail(err)
			}

			fmt.Println(g)
		},
	}

	return cmd
}
