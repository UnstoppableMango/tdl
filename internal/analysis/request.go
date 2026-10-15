package analysis

import (
	"slices"

	"github.com/unstoppablemango/tdl/ir"
	"github.com/unstoppablemango/tdl/plugin"
)

// Request builds the request for one target from a model: the model cut
// to the target's block and directives, its analysis, and the numbers the
// backend's Numbering gives its members. It returns what the analysis
// found wrong in the model, which a host reports before running the
// backend: a constraint that cannot hold, and a number the rules refuse.
func Request(m *ir.Model, desc plugin.Description, target, out string) (*plugin.Request, []*plugin.Diagnostic) {
	m = Scope(m, target)
	a := Analyze(m, target)
	diags := slices.Concat(Problems(m, a), Number(m, target, desc.Name, a, desc.Numbering))
	return &plugin.Request{Target: target, Model: m, Out: out, Analysis: a}, diags
}
