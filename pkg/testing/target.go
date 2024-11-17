package testing

import (
	"iter"

	tdl "github.com/unstoppablemango/tdl/pkg"
)

type MockTarget struct {
	ChooseFunc  func([]tdl.SinkGenerator) (tdl.SinkGenerator, error)
	PluginsFunc func() iter.Seq[tdl.Plugin]
	StringFunc  func() string
}

// Choose implements tdl.Target.
func (m *MockTarget) Choose(available []tdl.SinkGenerator) (tdl.SinkGenerator, error) {
	return m.ChooseFunc(available)
}

// Plugins implements tdl.Target.
func (m *MockTarget) Plugins() iter.Seq[tdl.Plugin] {
	return m.PluginsFunc()
}

// String implements tdl.Target.
func (m *MockTarget) String() string {
	return m.StringFunc()
}

func (m *MockTarget) WithChoose(
	fn func([]tdl.SinkGenerator) (tdl.SinkGenerator, error),
) *MockTarget {
	m.ChooseFunc = fn
	return m
}

func (m *MockTarget) WithPlugins(
	fn func() iter.Seq[tdl.Plugin],
) *MockTarget {
	m.PluginsFunc = fn
	return m
}

func (m *MockTarget) WithString(
	fn func() string,
) *MockTarget {
	m.StringFunc = fn
	return m
}

var _ tdl.Target = &MockTarget{}

func NewMockTarget() *MockTarget {
	return &MockTarget{
		ChooseFunc: func(g []tdl.SinkGenerator) (tdl.SinkGenerator, error) {
			panic("unimplemented")
		},
		PluginsFunc: func() iter.Seq[tdl.Plugin] {
			panic("unimplemented")
		},
		StringFunc: func() string {
			panic("unimplemented")
		},
	}
}
