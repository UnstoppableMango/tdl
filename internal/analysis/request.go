package analysis

import (
	"slices"

	"github.com/unstoppablemango/tdl/internal/host"
	"github.com/unstoppablemango/tdl/ir"
	"github.com/unstoppablemango/tdl/plugin"
)

// Request builds the request for one target from a model: the model cut
// to the target's block and directives, and its analysis, holding the
// numbers the backend's Numbering gives its members and, under the
// annotate directive, the model as TDL. It returns what the analysis
// found wrong in the model, which a host reports before running the
// backend: a constraint that cannot hold, and a number the rules refuse.
// A block directive in a deprecated spelling is renamed as [host.Rename]
// reads it.
func Request(m *ir.Model, desc plugin.Description, target, out string) (*plugin.Request, []*plugin.Diagnostic) {
	m = Scope(m, target)
	host.RenameBlocks(m, desc)
	a := Analyze(m, target)
	Carry(m, target, a)
	diags := slices.Concat(Problems(m, a), Number(m, target, desc.Name, a, desc.Numbering))
	return &plugin.Request{Target: target, Model: m, Out: out, Analysis: a}, diags
}
