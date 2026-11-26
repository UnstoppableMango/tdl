package analysis

import (
	"slices"

	"github.com/unstoppablemango/tdl/ir"
	"github.com/unstoppablemango/tdl/plugin"
)

// Request builds the request for one target from a model: the model cut
// to the target's block and directives, and its analysis, holding the
// numbers the backend's Numbering gives its members and, under the
// roundtrip directive or for a nondeterministic backend, the model as TDL. It returns what the analysis
// found wrong in the model, which a host reports before running the
// backend: a constraint that cannot hold, and a number the rules refuse.
func Request(m *ir.Model, desc plugin.Description, target, out string) (*plugin.Request, []*plugin.Diagnostic) {
	m = Scope(m, target)
	a := Analyze(m, target)
	Carry(m, target, a)
	// A nondeterministic backend writes from the model's TDL, not from
	// annotations, so it gets the source alone.
	if desc.Nondeterministic && a.Source == nil {
		a.Source = ModelSource(m, target)
	}
	diags := slices.Concat(Problems(m, a), Number(m, target, desc.Name, a, desc.Numbering))
	return &plugin.Request{Target: target, Model: m, Out: out, Analysis: a}, diags
}
