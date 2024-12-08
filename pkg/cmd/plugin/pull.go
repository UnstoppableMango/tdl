package plugin

import (
	"fmt"
	"os"

	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"
	"github.com/unstoppablemango/tdl/internal/util"
	"github.com/unstoppablemango/tdl/pkg/plugin"
)

func NewPull() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pull [PLUGIN]",
		Short: "Pull the specified plugin",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			log.SetLevel(log.DebugLevel)

			ctx := cmd.Context()
			err := plugin.Pull(ctx, args[0],
				plugin.WithProgress(func(f float64, err error) {
					if err != nil {
						fmt.Fprintln(os.Stderr, err)
					} else {
						fmt.Println(f)
					}
				}),
			)
			if err != nil {
				util.Fail(err)
			}

			fmt.Println("Done")
		},
	}

	return cmd
}
