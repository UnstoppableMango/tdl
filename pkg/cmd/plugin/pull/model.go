package pull

import (
	tea "github.com/charmbracelet/bubbletea"
	tdl "github.com/unstoppablemango/tdl/pkg"
	"github.com/unstoppablemango/tdl/pkg/progress"
)

type (
	ErrMsg error
)

type Model struct {
	prog   progress.Model
	plugin tdl.Plugin
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.prog, cmd = m.prog.Update(msg)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		}
	case tea.QuitMsg:
		return m, tea.Quit
	}

	return m, cmd
}

// View implements tea.Model.
func (m Model) View() string {
	return m.prog.View()
}

func NewModel(plugin tdl.Plugin) tea.Model {
	return Model{
		prog:   progress.NewModel(),
		plugin: plugin,
	}
}
