package testing

import (
	"context"

	tdl "github.com/unstoppablemango/tdl/pkg"
)

type MockPlugin struct {
	GeneratorFunc func(context.Context, tdl.Meta) (tdl.Generator, error)
	ToolFunc      func(context.Context, tdl.Meta) (tdl.Tool, error)
	SupportsFunc  func(tdl.Target) bool
	MetaValue     tdl.Meta
	StringFunc    func() string
}

// Meta implements tdl.GeneratorPlugin.
func (m *MockPlugin) Meta() tdl.Meta {
	if m.MetaValue == nil {
		panic("unimplemented")
	}

	return m.MetaValue
}

// Tool implements tdl.ToolPlugin.
func (p *MockPlugin) Tool(ctx context.Context, m tdl.Meta) (tdl.Tool, error) {
	if p.ToolFunc == nil {
		panic("unimplemented")
	}

	return p.ToolFunc(ctx, m)
}

// Generator implements tdl.Plugin.
func (p *MockPlugin) Generator(ctx context.Context, m tdl.Meta) (tdl.Generator, error) {
	if p.GeneratorFunc == nil {
		panic("unimplemented")
	}

	return p.GeneratorFunc(ctx, m)
}

func (p *MockPlugin) Supports(target tdl.Target) bool {
	if p.SupportsFunc == nil {
		panic("unimplemented")
	}

	return p.SupportsFunc(target)
}

// String implements tdl.Plugin.
func (p *MockPlugin) String() string {
	if p.StringFunc == nil {
		panic("unimplemented")
	}

	return p.StringFunc()
}

func (p *MockPlugin) WithGenerator(
	fn func(context.Context, tdl.Meta) (tdl.Generator, error),
) *MockPlugin {
	p.GeneratorFunc = fn
	return p
}

func (m *MockPlugin) WithString(
	fn func() string,
) *MockPlugin {
	m.StringFunc = fn
	return m
}
