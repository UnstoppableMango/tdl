package analysis_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/unstoppablemango/tdl/internal/analysis"
	"github.com/unstoppablemango/tdl/internal/sema"
	"github.com/unstoppablemango/tdl/ir"
	"github.com/unstoppablemango/tdl/parser"
	"github.com/unstoppablemango/tdl/plugin"
)

const numbered = `package shop

type Order {
  id: string
  pay: Payment
  note: string
  at: string
}

enum Payment {
  Card { last4: string }
  Cash { amount: int }
}

enum Status {
  Open
  Closed
}

target wire for shop {
  Order {
    reserved(2, "old")
    pay => oneof
    at => number(9)
  }
  Status.Closed => number(7)
}
`

// numbering is a rule set exercising every part of Numbering.
func numbering() *plugin.Numbering {
	low, high := int64(4), int64(5)
	return &plugin.Numbering{
		Fields:  &plugin.NumberRule{Max: 100, Reserved: []*ir.Range{{Low: &low, High: &high}}},
		Values:  &plugin.NumberRule{Max: 10},
		Reserve: "reserved",
		Inline:  "oneof",
	}
}

func lower(t *testing.T, src string) *ir.Model {
	t.Helper()
	file, err := parser.Parse("shop.tdl", strings.NewReader(src))
	if err != nil {
		t.Fatal(err)
	}
	model, diags := sema.Lower(file)
	if len(diags) > 0 {
		t.Fatalf("lowering: %v", diags)
	}
	return model
}

func view(t *testing.T, m *ir.Model, a *plugin.Analysis, name string) *plugin.DeclView {
	t.Helper()
	for i, d := range m.GetDecls() {
		if d.GetMeta().GetName() == name {
			return a.GetDecls()[i]
		}
	}
	t.Fatalf("no declaration %s", name)
	return nil
}

func TestNumber(t *testing.T) {
	m := lower(t, numbered)
	req, diags := analysis.Request(m, plugin.Description{Name: "wire", Numbering: numbering()}, "wire", "out")
	if len(diags) > 0 {
		t.Fatalf("diagnostics = %v", diags)
	}
	a := req.GetAnalysis()

	// id takes 1; 2 is reserved by the directive, so pay's variants take 3
	// and, past the reserved 4 and 5, 6; note takes 7; at is pinned.
	order := view(t, req.GetModel(), a, "Order")
	var got []int64
	for _, f := range order.GetFields() {
		got = append(got, f.GetNumber())
	}
	if want := []int64{1, 0, 7, 9}; !slices.Equal(got, want) {
		t.Errorf("Order's numbers = %v, want %v", got, want)
	}
	if got, want := order.GetFields()[1].GetInlined(), []int64{3, 6}; !slices.Equal(got, want) {
		t.Errorf("pay's inlined numbers = %v, want %v", got, want)
	}

	// A fielded enum's variants and their fields are numbered as fields.
	pay := view(t, req.GetModel(), a, "Payment")
	for i, want := range []int64{1, 2} {
		if got := pay.GetVariants()[i].GetNumber(); got != want {
			t.Errorf("Payment variant %d = %d, want %d", i, got, want)
		}
		if got := pay.GetVariants()[i].GetFields()[0].GetNumber(); got != 1 {
			t.Errorf("Payment variant %d's field = %d, want 1", i, got)
		}
	}

	// A fieldless enum's variants are numbered as values.
	status := view(t, req.GetModel(), a, "Status")
	if got := []int64{status.GetVariants()[0].GetNumber(), status.GetVariants()[1].GetNumber()}; !slices.Equal(got, []int64{1, 7}) {
		t.Errorf("Status's numbers = %v, want [1 7]", got)
	}
}

func TestNumberNothingWithoutRules(t *testing.T) {
	m := lower(t, numbered)
	req, diags := analysis.Request(m, plugin.Description{Name: "wire"}, "wire", "out")
	if len(diags) > 0 {
		t.Fatalf("diagnostics = %v", diags)
	}
	for _, f := range view(t, req.GetModel(), req.GetAnalysis(), "Order").GetFields() {
		if f.GetNumber() != 0 || len(f.GetInlined()) > 0 {
			t.Errorf("a backend without Numbering got numbers: %v", f)
		}
	}
}

// Each refused pin is one error at the pin, and leaves its space
// unnumbered.
func TestNumberRefusesPins(t *testing.T) {
	for name, tc := range map[string]struct{ pin, want string }{
		"reserved by the declaration": {"id => number(2)", "Order.id is numbered 2, which Order reserves"},
		"reserved by the rule":        {"id => number(4)", "Order.id is numbered 4, which wire reserves (4 to 5)"},
		"out of range":                {"id => number(101)", "Order.id is numbered 101, and wire numbers run from 1 to 100"},
		"shared":                      {"id => number(9)", "Order.id and Order.at are both numbered 9"},
	} {
		t.Run(name, func(t *testing.T) {
			src := strings.Replace(numbered, "at => number(9)", "at => number(9)\n    "+tc.pin, 1)
			m := lower(t, src)
			req, diags := analysis.Request(m, plugin.Description{Name: "wire", Numbering: numbering()}, "wire", "out")
			if len(diags) != 1 {
				t.Fatalf("diagnostics = %v, want one", diags)
			}
			d := diags[0]
			if d.GetSeverity() != plugin.Severity_SEVERITY_ERROR || d.GetMessage() != tc.want {
				t.Errorf("diagnostic = %v %q, want an error %q", d.GetSeverity(), d.GetMessage(), tc.want)
			}
			if d.GetPosition().GetLine() == 0 {
				t.Error("the diagnostic has no position")
			}
			for _, f := range view(t, req.GetModel(), req.GetAnalysis(), "Order").GetFields() {
				if f.GetNumber() != 0 {
					t.Errorf("a refused space was numbered: %v", f)
				}
			}
		})
	}
}
