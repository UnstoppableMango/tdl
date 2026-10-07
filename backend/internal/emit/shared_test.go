package emit_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/unstoppablemango/tdl/backend/internal/emit"
	"github.com/unstoppablemango/tdl/backend/internal/irtest"
	"github.com/unstoppablemango/tdl/ir"
)

func TestLength(t *testing.T) {
	i := func(n int64) *int64 { return &n }
	rng := func(lo, hi *int64) *ir.Literal {
		return &ir.Literal{Kind: ir.LiteralKind_LITERAL_KIND_RANGE, Range: &ir.Range{Low: lo, High: hi}}
	}
	for _, tt := range []struct {
		name      string
		in        *ir.Literal
		low, high *int64
		err       string
	}{
		{"exact", &ir.Literal{Kind: ir.LiteralKind_LITERAL_KIND_INT, Text: "16"}, i(16), i(16), ""},
		{"closed", rng(i(3), i(254)), i(3), i(254), ""},
		{"open above", rng(i(1), nil), i(1), nil, ""},
		{"open below", rng(nil, i(8)), nil, i(8), ""},
		{"empty", rng(nil, nil), nil, nil, "neither end"},
		{"inverted", rng(i(5), i(2)), nil, nil, "can never hold"},
		{"too wide", &ir.Literal{Kind: ir.LiteralKind_LITERAL_KIND_INT, Text: "99999999999999999999"}, nil, nil, "does not fit"},
		{"a string", irtest.Text("3"), nil, nil, "its length is"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			low, high, err := emit.Length(tt.in)
			if tt.err != "" {
				if err == nil || !strings.Contains(err.Error(), tt.err) {
					t.Fatalf("Length = %v, want an error containing %q", err, tt.err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			eq := func(a, b *int64) bool { return (a == nil) == (b == nil) && (a == nil || *a == *b) }
			if !eq(low, tt.low) || !eq(high, tt.high) {
				t.Errorf("Length = %v, %v; want %v, %v", low, high, tt.low, tt.high)
			}
		})
	}
}

func TestPattern(t *testing.T) {
	re := func(s string) *ir.Literal { return &ir.Literal{Kind: ir.LiteralKind_LITERAL_KIND_REGEX, Text: s} }
	if _, err := emit.Pattern(re(`^[^@]+@[^@]+$`)); err != nil {
		t.Errorf("an RE2 pattern: %v", err)
	}
	// Lookahead is ECMAScript and PCRE, not RE2.
	if _, err := emit.Pattern(re(`^(?=a)`)); err == nil || !strings.Contains(err.Error(), "RE2 refuses") {
		t.Errorf("a lookahead: %v, want RE2 to refuse it", err)
	}
	if _, err := emit.Pattern(irtest.Text("a")); err == nil {
		t.Error("a string literal is not a pattern")
	}
}

func TestDiscriminant(t *testing.T) {
	disc := func(target string) []*ir.Directive {
		return []*ir.Directive{{Name: "discriminant", Target: target, Args: []*ir.Literal{irtest.Text("type")}}}
	}
	enum := &ir.Decl{Meta: &ir.Meta{Name: "Payment"}, Node: &ir.Decl_Enumeration{Enumeration: &ir.Enum{}}}

	b := irtest.New("shop")
	if got := session(b).Discriminant(enum); got != emit.DefaultDiscriminant {
		t.Errorf("no directive: %q, want %q", got, emit.DefaultDiscriminant)
	}

	b.Model.Targets = append(b.Model.Targets, &ir.TargetBlock{Meta: &ir.Meta{Name: "x"}, Directives: disc("x")})
	if got := session(b).Discriminant(enum); got != "type" {
		t.Errorf("the block's: %q, want type", got)
	}

	enum.Directives = []*ir.Directive{{Name: "discriminant", Target: "x", Args: []*ir.Literal{irtest.Text("tag")}}}
	if got := session(b).Discriminant(enum); got != "tag" {
		t.Errorf("the enum's over the block's: %q, want tag", got)
	}

	v := &ir.Variant{Meta: &ir.Meta{Name: "Card"}, Directives: []*ir.Directive{{Name: "name", Target: "x", Args: []*ir.Literal{irtest.Text("CardPayment")}}}}
	if got := emit.Tag(v); got != "Card" {
		t.Errorf("Tag = %q, want the model's name", got)
	}
}

func TestPlanInterfaces(t *testing.T) {
	b := irtest.New("shop")
	b.Own(&ir.Decl{Meta: &ir.Meta{Name: "Order"}, Node: &ir.Decl_Structure{Structure: &ir.Struct{}}})
	b.Own(&ir.Decl{Meta: &ir.Meta{Name: "Fax"}, Node: &ir.Decl_Structure{Structure: &ir.Struct{}}})
	b.Class("Timestamped")
	b.Class("Auditable", "Timestamped")
	loop := b.Class("Loop")
	loop.GetClass().RequiresClasses = append(loop.GetClass().RequiresClasses, &ir.ClassRef{Class: b.Ref("Loop")})
	b.Class("Skipped")
	b.Satisfies("Auditable", "Order", "Fax")
	b.Satisfies("Timestamped", "Order")

	s := session(b)
	plan := s.PlanInterfaces(emit.InterfaceRules{
		Generates: func(d *ir.Decl) bool { return d.GetMeta().GetName() != "Skipped" },
		Carry: func(target, class *ir.Decl) error {
			if target.GetMeta().GetName() == "Fax" {
				return emit.Unsupported(nil, "no fax")
			}
			return nil
		},
	})

	idx := func(name string) int32 { return b.Ref(name).GetIndex() }
	if !plan.Classes[idx("Auditable")] || !plan.Classes[idx("Timestamped")] {
		t.Error("Auditable and Timestamped should be generated")
	}
	if plan.Classes[idx("Skipped")] || plan.Classes[idx("Loop")] {
		t.Error("Skipped and Loop should not be generated")
	}
	if !plan.Cyclic[idx("Loop")] {
		t.Error("Loop reaches itself")
	}
	if got, want := plan.Implements[idx("Order")], []int32{idx("Timestamped"), idx("Auditable")}; !slices.Equal(got, want) {
		t.Errorf("Order implements %v, want %v", got, want)
	}
	if got := plan.Implements[idx("Fax")]; len(got) != 0 {
		t.Errorf("Fax implements %v, want nothing", got)
	}
	if len(s.Diags) != 1 {
		t.Errorf("warnings = %v, want the one Carry refused", s.Diags)
	}

	inst := func(arg *ir.ID, params ...*ir.Param) *ir.Instance {
		return &ir.Instance{Meta: &ir.Meta{}, Params: params, Class: &ir.ClassRef{Class: b.Ref("Auditable"), Args: []*ir.ID{arg}}}
	}
	for _, tt := range []struct {
		name string
		inst *ir.Instance
		want emit.Refusal
		typ  string
	}{
		{"own", inst(b.Named("Order")), emit.Implemented, ""},
		{"conditional", inst(b.Param("T", 0), irtest.Params("T")...), emit.Conditional, ""},
		{"foreign", inst(b.Extern("acme.Money")), emit.ForeignType, "acme.Money"},
		{"prelude", inst(b.Named("string")), emit.PreludeType, "string"},
	} {
		if got, typ := plan.Instance(b.Model, tt.inst); got != tt.want || typ != tt.typ {
			t.Errorf("%s: Instance = %v, %q; want %v, %q", tt.name, got, typ, tt.want, tt.typ)
		}
	}
}
