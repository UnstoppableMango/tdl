package pull

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/log"
	tdl "github.com/unstoppablemango/tdl/pkg"
	"github.com/unstoppablemango/tdl/pkg/plugin"
	"github.com/unstoppablemango/tdl/pkg/progress"
)

type (
	ErrMsg error
)

type Model struct {
	prog   progress.Model
	sub    chan *progress.Event
	errs   chan error
	plugin tdl.Plugin
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return tea.Batch(m.pull, m.listen)
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.prog, cmd = m.prog.Update(msg)

	switch msg.(type) {
	case progress.ProgressMsg:
		return m, tea.Batch(cmd, m.listen)
	case ErrMsg:
		log.Errorf("err: %s", msg)
		return m, tea.Batch(cmd, m.listen)
	default:
		return m, cmd
	}
}

// View implements tea.Model.
func (m Model) View() string {
	return m.prog.View()
}

func NewModel(plugin tdl.Plugin) tea.Model {
	return Model{
		prog:   progress.NewModel(),
		sub:    make(chan *progress.Event),
		errs:   make(chan error),
		plugin: plugin,
	}
}

func (m Model) pull() tea.Msg {
	handler := progress.ChannelHandler(m.sub, m.errs)
	err := plugin.Pull(context.Background(), m.plugin,
		plugin.WithProgress(handler),
	)
	if err != nil {
		m.errs <- err
	}

	return tea.Quit()
}

func (m Model) listen() tea.Msg {
	select {
	case err := <-m.errs:
		return err
	case e := <-m.sub:
		return progress.ToMsg(e)
	}
}
