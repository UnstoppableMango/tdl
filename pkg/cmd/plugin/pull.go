package plugin

import (
	"context"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"
	"github.com/unstoppablemango/tdl/internal/util"
	"github.com/unstoppablemango/tdl/pkg/logging"
	"github.com/unstoppablemango/tdl/pkg/plugin"
	"github.com/unstoppablemango/tdl/pkg/progress"
)

func NewPull() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pull [PLUGIN]",
		Short: "Pull the specified plugin",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			logging.Init()
			ctx := cmd.Context()
			prog := tea.NewProgram(progress.NewModel())
			errs := make(chan error)

			go pull(ctx, args[0], prog, errs)

			log.Debug("starting tea app")
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
	log.Debugf("pulling %s", name)
	err := plugin.PullToken(ctx, name,
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
