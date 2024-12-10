package logging

import (
	"context"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/log"
	"github.com/unmango/go/option"
)

type (
	LogMsg  []byte
	DoneMsg struct{}
)

type Model struct {
	content tea.Model
	msgs    []LogMsg

	ctx    context.Context
	sub    chan []byte
	cancel context.CancelFunc
}

type ShellOption func(*Model)

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return m.listen
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.content, cmd = m.content.Update(msg)

	switch msg := msg.(type) {
	case LogMsg:
		m.msgs = append(m.msgs, msg)
		cmd = tea.Batch(cmd, m.listen)
	}

	return m, cmd
}

// View implements tea.Model.
func (m Model) View() string {
	if len(m.msgs) == 0 {
		return m.content.View()
	}

	msgs := []string{}
	for _, msg := range m.msgs {
		msgs = append(msgs, string(msg))
	}

	return strings.Join([]string{
		strings.Join(msgs, "\n"),
		m.content.View(),
	}, "\n")
}

func (m Model) listen() tea.Msg {
	select {
	case <-m.ctx.Done():
		return DoneMsg{}
	case msg := <-m.sub:
		return LogMsg(msg)
	}
}

func NewShell(content tea.Model, options ...ShellOption) Model {
	m := Model{
		content: content,
		msgs:    []LogMsg{},
		cancel:  func() {},
		sub:     make(chan []byte),
	}
	option.ApplyAll(&m, options)

	if m.ctx == nil {
		m.ctx, m.cancel = context.WithTimeout(
			context.Background(),
			time.Second*30,
		)
	}

	return m
}

// WithLogger sets the output of the specified logger to print above the given content.
// I'm not actually sure if this works yet because I'm bad at concurrency.
func WithLogger(log *log.Logger) ShellOption {
	return func(m *Model) {
		log.SetOutput(channel(m.sub))
	}
}

func WithContext(ctx context.Context) ShellOption {
	return func(m *Model) {
		m.ctx = ctx
	}
}
