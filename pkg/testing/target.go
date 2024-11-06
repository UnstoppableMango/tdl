package testing

import (
	"iter"

	tdl "github.com/unstoppablemango/tdl/pkg"
)

type MockTarget struct {
	ChooseParams struct {
		Available []tdl.Generator
	}
	ChooseResult struct {
		Generator tdl.Generator
		Err       error
	}

	PluginsResult iter.Seq[tdl.Plugin]

	StringResult string
}

// Choose implements tdl.Target.
func (m *MockTarget) Choose(available []tdl.Generator) (tdl.Generator, error) {
	m.ChooseParams = struct{ Available []tdl.Generator }{
		Available: available,
	}

	return m.ChooseResult.Generator, m.ChooseResult.Err
}

// Plugins implements tdl.Target.
func (m *MockTarget) Plugins() iter.Seq[tdl.Plugin] {
	return m.PluginsResult
}

// String implements tdl.Target.
func (m *MockTarget) String() string {
	return m.StringResult
}

var _ tdl.Target = &MockTarget{}
