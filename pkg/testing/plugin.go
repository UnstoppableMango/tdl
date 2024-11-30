package testing

import (
	"context"

	tdl "github.com/unstoppablemango/tdl/pkg"
)

type MockPlugin struct {
	GeneratorFunc func(tdl.Target) (tdl.Generator, error)
	StringFunc    func() string
}

// Generator implements tdl.Plugin.
func (m *MockPlugin) Generator(ctx context.Context, t tdl.Target) (tdl.Generator, error) {
	if m.GeneratorFunc == nil {
		panic("unimplemented")
	}

	return m.GeneratorFunc(t)
}

// String implements tdl.Plugin.
func (m *MockPlugin) String() string {
	return m.StringFunc()
}

func (m *MockPlugin) WithGenerator(
	fn func(t tdl.Target) (tdl.Generator, error),
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
