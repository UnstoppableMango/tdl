package pull

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"
	tdl "github.com/unstoppablemango/tdl/pkg"
	"github.com/unstoppablemango/tdl/pkg/plugin"
	"github.com/unstoppablemango/tdl/pkg/progress"
)

type Model struct {
	prog   progress.Model
	plugin tdl.Plugin
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return m.pull
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	panic("unimplemented")
}

// View implements tea.Model.
func (m Model) View() string {
	return m.prog.View()
}

func NewModel() tea.Model {
	return Model{
		prog: progress.NewModel(),
	}
}

func (m Model) pull() tea.Msg {
	var (
		ctx    = context.TODO()
		events = make(chan *progress.Event)
		errs   = make(chan error)
	)

	handler := progress.ChannelHandler(events, errs)
	go func() {
		err := plugin.Pull(ctx, m.plugin,
			plugin.WithProgress(handler),
		)
		if err != nil {
			errs <- err
		}
	}()

	return nil
}

func batchChan(ctx context.Context, msgs <-chan tea.Msg) tea.Cmd {
	return func() tea.Msg {
		select {
		case <-ctx.Done():
			return nil
		case <-msgs:
			return batchChan(ctx, msgs)
		}
	}
}
