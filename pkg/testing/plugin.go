package testing

import (
	"context"

	tdl "github.com/unstoppablemango/tdl/pkg"
)

type MockPlugin struct {
	GeneratorFunc     func(tdl.Target) (tdl.Generator, error)
	SinkGeneratorFunc func(tdl.Target) (tdl.SinkGenerator, error)
	StringFunc        func() string
}

// Generator implements tdl.Plugin.
func (m *MockPlugin) Generator(ctx context.Context, t tdl.Target) (tdl.Generator, error) {
	if m.GeneratorFunc == nil {
		panic("unimplemented")
	}

	return m.GeneratorFunc(t)
}

// SinkGenerator implements tdl.Plugin.
func (m *MockPlugin) SinkGenerator(t tdl.Target) (tdl.SinkGenerator, error) {
	if m.SinkGeneratorFunc == nil {
		panic("unimplemented")
	}

	return m.SinkGeneratorFunc(t)
}

// String implements tdl.Plugin.
func (m *MockPlugin) String() string {
	return m.StringFunc()
}

func (m *MockPlugin) WithGenerator(
	fn func(t tdl.Target) (tdl.SinkGenerator, error),
) *MockPlugin {
	m.SinkGeneratorFunc = fn
	return m
}

func (m *MockPlugin) WithString(
	fn func() string,
) *MockPlugin {
	m.StringFunc = fn
	return m
}

var _ tdl.Plugin = &MockPlugin{}
