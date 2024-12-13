package plugin

import (
	"context"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"github.com/unstoppablemango/tdl/internal/util"
	tdl "github.com/unstoppablemango/tdl/pkg"
	"github.com/unstoppablemango/tdl/pkg/cmd/plugin/pull"
	"github.com/unstoppablemango/tdl/pkg/logging"
	"github.com/unstoppablemango/tdl/pkg/plugin"
	"github.com/unstoppablemango/tdl/pkg/progress"
)

func NewPull() *cobra.Command {
	return &cobra.Command{
		Use:   "pull [PLUGIN]",
		Short: "Pull the specified plugin",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			logging.Init()
			p, err := plugin.ParseToken(args[0])
			if err != nil {
				util.Fail(err)
			}

			if _, ok := os.LookupEnv("DISABLE_TUI"); ok {
				err = pullTerm(cmd.Context(), p)
			} else {
				err = pullTui(cmd.Context(), p)
			}
			if err != nil {
				util.Fail(err)
			}

			fmt.Println("Done")
		},
	}
}

func pullTui(ctx context.Context, p tdl.Plugin) error {
	prog := tea.NewProgram(pull.NewModel(p))
	// logging.LogToProgram(prog)

	sub := plugin.Subscribe(p, progress.TeaHandler(prog))
	defer sub()

	go func() {
		err := plugin.Pull(ctx, p)
		if err != nil {
			prog.Println(err)
		}

		prog.Send(tea.Quit())
	}()

	if _, err := prog.Run(); err != nil {
		return err
	} else {
		return nil
	}
}

func pullTerm(ctx context.Context, p tdl.Plugin) error {
	return plugin.Pull(ctx, p) // TODO: Progress?
}
