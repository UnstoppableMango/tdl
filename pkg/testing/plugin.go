package testing

import (
	tdl "github.com/unstoppablemango/tdl/pkg"
)

type MockPlugin struct {
	GeneratorFunc func(tdl.Target) (tdl.SinkGenerator, error)
	StringFunc    func() string
}

// Generator implements tdl.Plugin.
func (m *MockPlugin) Generator(t tdl.Target) (tdl.SinkGenerator, error) {
	return m.GeneratorFunc(t)
}

// String implements tdl.Plugin.
func (m *MockPlugin) String() string {
	return m.StringFunc()
}

func (m *MockPlugin) WithGenerator(
	fn func(t tdl.Target) (tdl.SinkGenerator, error),
) *MockPlugin {
	m.GeneratorFunc = fn
	return m
}

func (m *MockPlugin) WithString(
	fn func() string,
) *MockPlugin {
	m.StringFunc = fn
	return m
}

var _ tdl.Plugin = &MockPlugin{}

func NewMockPlugin() *MockPlugin {
	return &MockPlugin{
		GeneratorFunc: func(t tdl.Target) (tdl.SinkGenerator, error) {
			panic("unimplemented")
		},
		StringFunc: func() string {
			panic("unimplemented")
		},
	}
}
