package progress

import (
	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
)

const (
	padding  = 2
	maxWidth = 80
)

type (
	ProgressMsg float64
	DoneMsg     struct{}
)

type Model struct {
	progress progress.Model
	percent  float64
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.progress.Width = msg.Width - padding*2 - 4
		if m.progress.Width > maxWidth {
			m.progress.Width = maxWidth
		}
	case ProgressMsg:
		m.percent = float64(msg)
		if m.percent >= 1.0 {
			m.percent = 1.0
			return m, tea.Quit
		}
	case DoneMsg:
		m.percent = 1.0
	case tea.QuitMsg:
		m.percent = 1.0
		return m, tea.Quit
	}

	return m, nil
}

// View implements tea.Model.
func (m Model) View() string {
	return m.progress.ViewAs(m.percent) + "\n"
}

func NewModel() Model {
	return Model{
		progress: progress.New(),
	}
}

func ToMsg(e *Event) ProgressMsg {
	return ProgressMsg(e.Percent())
}

func TeaHandler(prog *tea.Program) HandlerFunc {
	return func(e *Event, err error) {
		prog.Send(e.ProgressMsg())
	}
}

func Done() tea.Msg {
	return DoneMsg{}
}
