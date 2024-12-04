package testing

import (
	"github.com/unmango/go/iter"
	tdl "github.com/unstoppablemango/tdl/pkg"
)

type MockTarget struct {
	ChooseFunc func(iter.Seq[tdl.Plugin]) (tdl.Plugin, error)
	MetaFunc   func() tdl.Meta
	StringFunc func() string
}

// Meta implements tdl.Target.
func (m *MockTarget) Meta() tdl.Meta {
	if m.MetaFunc == nil {
		panic("unimplemented")
	}

	return m.MetaFunc()
}

// Tool implements tdl.Target.
func (m *MockTarget) Choose(available iter.Seq[tdl.Plugin]) (tdl.Plugin, error) {
	if m.ChooseFunc == nil {
		panic("unimplemented")
	}

	return m.ChooseFunc(available)
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
	fn func(iter.Seq[tdl.Plugin]) (tdl.Plugin, error),
) *MockTarget {
	m.ChooseFunc = fn
	return m
}

var _ tdl.Target = &MockTarget{}
