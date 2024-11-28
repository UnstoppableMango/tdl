package plugin

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/unstoppablemango/tdl/pkg/plugin"
)

func NewList() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List available plugins",
		Aliases: []string{"ls"},
		Args:    cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			for _, p := range plugin.Static() {
				fmt.Println(plugin.Unwrap(p))
			}
		},
	}

	return cmd
}
