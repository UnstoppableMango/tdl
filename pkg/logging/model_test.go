package logging_test

import (
	"context"
	"io"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/log"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/unstoppablemango/tdl/pkg/logging"
)

type contentStub string

// Init implements tea.Model.
func (c contentStub) Init() tea.Cmd { return nil }

// Update implements tea.Model.
func (c contentStub) Update(tea.Msg) (tea.Model, tea.Cmd) { return c, nil }

// View implements tea.Model.
func (c contentStub) View() string { return string(c) }

var _ = Describe("Model", func() {
	It("should print the content", func() {
		m := logging.NewShell(contentStub("testing"))

		view := m.View()

		Expect(view).To(Equal("testing"))
	})

	It("should print messages above the content", func() {
		m := logging.NewShell(contentStub("testing"))
		m, _ = m.Update(logging.LogMsg([]byte("blah")))

		view := m.View()

		Expect(view).To(Equal("blah\ntesting"))
	})

	It("should separate multiple messages with a newline", func() {
		m := logging.NewShell(contentStub("testing"))
		m, _ = m.Update(logging.LogMsg([]byte("blah1")))
		m, _ = m.Update(logging.LogMsg([]byte("blah2")))

		view := m.View()

		Expect(view).To(Equal("blah1\nblah2\ntesting"))
	})

	It("should display messages from the given logger", Pending, func(ctx context.Context) {
		// Concurrency is hard
		log := log.New(io.Discard)
		m := logging.NewShell(contentStub("testing"),
			logging.WithLogger(log),
			logging.WithContext(ctx),
		)
		_ = m.Init()
		log.Fatal("blah")

		view := m.View()

		Expect(view).To(Equal("blah\ntesting"))
	})
})
