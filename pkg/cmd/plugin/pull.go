package plugin

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"github.com/unstoppablemango/tdl/internal/util"
	"github.com/unstoppablemango/tdl/pkg/cmd/plugin/pull"
	"github.com/unstoppablemango/tdl/pkg/logging"
	"github.com/unstoppablemango/tdl/pkg/plugin"
)

func NewPull() *cobra.Command {
	return &cobra.Command{
		Use:   "pull [PLUGIN]",
		Short: "Pull the specified plugin",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			logging.Init()
			plugin, err := plugin.ParseToken(args[0])
			if err != nil {
				util.Fail(err)
			}

			prog := tea.NewProgram(pull.NewModel(plugin))
			if _, err := prog.Run(); err != nil {
				util.Fail(err)
			}

			fmt.Println("Done")
		},
	}
}
