package plugin

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/unstoppablemango/tdl/pkg/logging"
	"github.com/unstoppablemango/tdl/pkg/plugin"
)

func NewList() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List available plugins",
		Aliases: []string{"ls"},
		Args:    cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			logging.Init()
			for p := range plugin.Static() {
				for _, pi := range plugin.Unwrap(p) {
					fmt.Println(pi)
				}
			}
		},
	}

	return cmd
}
