package analysis_test

import (
	"testing"

	"github.com/unstoppablemango/tdl/internal/analysis"
	"github.com/unstoppablemango/tdl/plugin"
)

const constrained = `package shop

type Code: string where { length(2..8) matches(/^[A-Z]+$/) }
type Short: Code where { length(3) }

type Order {
  ref: string where { matches(/(?=x)/) length(..5) }
}

target wire for shop {}
`

func TestConstraintViews(t *testing.T) {
	m := lower(t, constrained)
	req, diags := analysis.Request(m, plugin.Description{Name: "wire"}, "wire", "out")
	if len(diags) > 0 {
		t.Fatalf("diagnostics = %v", diags)
	}
	a := req.GetAnalysis()

	code := view(t, req.GetModel(), a, "Code").GetConstraints()
	if len(code) != 2 {
		t.Fatalf("Code's constraints = %v", code)
	}
	if code[0].GetSource() != "length(2..8)" || code[0].GetLength().GetLow() != 2 || code[0].GetLength().GetHigh() != 8 {
		t.Errorf("length = %v", code[0])
	}
	if code[1].GetSource() != "matches(/^[A-Z]+$/)" || !code[1].GetRe2() {
		t.Errorf("matches = %v", code[1])
	}

	// An integer length is both ends.
	var own *plugin.ConstraintView
	for _, c := range view(t, req.GetModel(), a, "Short").GetConstraints() {
		if c.GetSource() == "length(3)" {
			own = c
		}
	}
	if l := own.GetLength(); l.GetLow() != 3 || l.GetHigh() != 3 {
		t.Errorf("length(3) = %v", l)
	}

	// A lookahead is not RE2, and is not a problem; an open end stays
	// unset.
	ref := view(t, req.GetModel(), a, "Order").GetFields()[0].GetConstraints()
	if ref[0].GetRe2() || ref[0].GetProblem() != "" {
		t.Errorf("lookahead = %v", ref[0])
	}
	if l := ref[1].GetLength(); l.Low != nil || l.GetHigh() != 5 {
		t.Errorf("length(..5) = %v", l)
	}
}

// A length that can never hold is one error at the constraint, even
// where a newtype inherits it.
func TestConstraintProblems(t *testing.T) {
	m := lower(t, `package shop

type Code: string where { length(8..2) }
type Short: Code

target wire for shop {}
`)
	_, diags := analysis.Request(m, plugin.Description{Name: "wire"}, "wire", "out")
	if len(diags) != 1 {
		t.Fatalf("diagnostics = %v, want one", diags)
	}
	d := diags[0]
	if d.GetSeverity() != plugin.Severity_SEVERITY_ERROR || d.GetMessage() != "length 8..2 can never hold" || d.GetPosition().GetLine() != 3 {
		t.Errorf("diagnostic = %v", d)
	}
}
