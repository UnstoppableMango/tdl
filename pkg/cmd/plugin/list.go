package plugin

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/unstoppablemango/tdl/pkg/plugin"
	"github.com/unstoppablemango/tdl/pkg/target"
)

func NewList() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list [TARGET]",
		Short:   "List available plugins",
		Aliases: []string{"ls"},
		Args:    cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) > 0 {
				t, err := target.Parse(args[0])
				if err != nil {
					fmt.Fprintln(os.Stderr, err)
					os.Exit(1)
				}

				for p := range t.Plugins() {
					fmt.Println(plugin.Unwrap(p))
				}
			}
		},
	}

	return cmd
}