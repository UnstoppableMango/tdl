package testing

import (
	"github.com/unmango/go/iter"
	tdl "github.com/unstoppablemango/tdl/pkg"
)

type MockTarget struct {
	ChooseFunc    func([]tdl.SinkGenerator) (tdl.SinkGenerator, error)
	GeneratorFunc func(iter.Seq[tdl.Plugin]) (tdl.Generator, error)
	PluginsFunc   func() iter.Seq[tdl.Plugin]
	StringFunc    func() string
}

// Generator implements tdl.Target.
func (m *MockTarget) Generator(available iter.Seq[tdl.Plugin]) (tdl.Generator, error) {
	if m.GeneratorFunc == nil {
		panic("unimplemented")
	}

	return m.GeneratorFunc(available)
}

// Choose implements tdl.Target.
func (m *MockTarget) Choose(available []tdl.SinkGenerator) (tdl.SinkGenerator, error) {
	if m.ChooseFunc == nil {
		panic("unimplemented")
	}

	return m.ChooseFunc(available)
}

// Plugins implements tdl.Target.
func (m *MockTarget) Plugins() iter.Seq[tdl.Plugin] {
	if m.PluginsFunc == nil {
		panic("unimplemented")
	}

	return m.PluginsFunc()
}

// String implements tdl.Target.
func (m *MockTarget) String() string {
	if m.StringFunc == nil {
		panic("unimplemented")
	}

	return m.StringFunc()
}

func (m *MockTarget) WithChoose(
	fn func([]tdl.Generator) (tdl.Generator, error),
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
