package testing

import (
	"github.com/unmango/go/iter"
	tdl "github.com/unstoppablemango/tdl/pkg"
)

type MockTarget struct {
	GeneratorFunc func(iter.Seq[tdl.Plugin]) (tdl.Generator, error)
	StringFunc    func() string
}

// Generator implements tdl.Target.
func (m *MockTarget) Generator(available iter.Seq[tdl.Plugin]) (tdl.Generator, error) {
	if m.GeneratorFunc == nil {
		panic("unimplemented")
	}

	return m.GeneratorFunc(available)
}

// String implements tdl.Target.
func (m *MockTarget) String() string {
	if m.StringFunc == nil {
		panic("unimplemented")
	}

	return m.StringFunc()
}

func (m *MockTarget) WithString(
	fn func() string,
) *MockTarget {
	m.StringFunc = fn
	return m
}

func (m *MockTarget) WithGenerator(
	fn func(iter.Seq[tdl.Plugin]) (tdl.Generator, error),
) *MockTarget {
	m.GeneratorFunc = fn
	return m
}

var _ tdl.Target = &MockTarget{}
