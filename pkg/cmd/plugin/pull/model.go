package pull

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/unstoppablemango/tdl/pkg/progress"
)

type model struct {
	progress.Model
}

func NewModel() (tea.Model, error) {
	return &model{}, nil
}
