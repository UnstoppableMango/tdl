package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/unstoppablemango/tdl/internal/util"
	"github.com/unstoppablemango/tdl/pkg/plugin"
	"github.com/unstoppablemango/tdl/pkg/target"
)

func NewWhich() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "which [NAMEISH]",
		Short: "Print the name of the closest matching generator",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			t, err := target.Parse(args[0])
			if err != nil {
				util.Fail(err)
			}

			for p := range plugin.Static() {
				g, err := p.Generator(cmd.Context(), t)
				if err != nil {
					util.Fail(err)
				}

				fmt.Println(g)
			}
		},
	}

	return cmd
}
