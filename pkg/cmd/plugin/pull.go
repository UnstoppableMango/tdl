package plugin

import (
	"context"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"github.com/unstoppablemango/tdl/internal/util"
	"github.com/unstoppablemango/tdl/pkg/plugin"
	"github.com/unstoppablemango/tdl/pkg/progress"
)

func NewPull() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pull [PLUGIN]",
		Short: "Pull the specified plugin",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			prog := tea.NewProgram(progress.NewModel())
			ctx := cmd.Context()
			errs := make(chan error)

			go pull(ctx, args[0], prog, errs)

			if _, err := prog.Run(); err != nil {
				util.Fail(err)
			}
			if len(errs) > 0 {
				for err := range errs {
					util.Fail(err)
				}
			}

			fmt.Println("Done")
		},
	}

	return cmd
}

func pull(ctx context.Context, name string, prog *tea.Program, errs chan<- error) {
	err := plugin.Pull(ctx, name,
		plugin.WithProgress(func(f float64, err error) {
			if err != nil {
				errs <- err
			} else {
				prog.Send(progress.ProgressMsg(f))
			}
		}),
	)
	if err != nil {
		errs <- err
	}
}
