package gen_test

import (
	"strings"
	"testing"

	"github.com/unstoppablemango/tdl/internal/gen"
	"github.com/unstoppablemango/tdl/ir"
	"github.com/unstoppablemango/tdl/plugin"
)

func spec() plugin.Description {
	return plugin.Description{
		Name: "t",
		Directives: []*plugin.DirectiveSpec{
			{
				Name:     "tag",
				MinArgs:  1,
				MaxArgs:  1,
				ArgKinds: []ir.LiteralKind{ir.LiteralKind_LITERAL_KIND_STRING},
			},
			{Name: "slice"},
			{Name: "anything", MinArgs: 1, MaxArgs: -1},
		},
	}
}

func modelWith(directives ...*ir.Directive) *ir.Model {
	return &ir.Model{
		Decls: []*ir.Decl{{
			Meta:       &ir.Meta{Name: "Order"},
			Directives: directives,
		}},
	}
}

func str(text string) *ir.Literal {
	return &ir.Literal{Kind: ir.LiteralKind_LITERAL_KIND_STRING, Text: text}
}

func TestDirectivesThatFit(t *testing.T) {
	model := modelWith(
		&ir.Directive{Name: "tag", Target: "t", Args: []*ir.Literal{str("x")}},
		&ir.Directive{Name: "slice", Target: "t"},
		&ir.Directive{Name: "anything", Target: "t", Args: []*ir.Literal{str("a"), str("b"), str("c")}},
	)
	if problems := gen.CheckDirectives("t", model, spec()); len(problems) != 0 {
		t.Errorf("problems = %v", problems)
	}
}

func TestDirectiveArity(t *testing.T) {
	tests := []struct {
		name string
		args []*ir.Literal
		want string
	}{
		{"too few", nil, "at least 1"},
		{"too many", []*ir.Literal{str("a"), str("b")}, "at most 1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := modelWith(&ir.Directive{Name: "tag", Target: "t", Args: tt.args})
			problems := gen.CheckDirectives("t", model, spec())
			if len(problems) != 1 || problems[0].Warning {
				t.Fatalf("problems = %v", problems)
			}
			if !strings.Contains(problems[0].Message, tt.want) {
				t.Errorf("message = %q, want it to mention %q", problems[0].Message, tt.want)
			}
		})
	}
}

func TestDirectiveArgumentKind(t *testing.T) {
	model := modelWith(&ir.Directive{
		Name:   "tag",
		Target: "t",
		Args:   []*ir.Literal{{Kind: ir.LiteralKind_LITERAL_KIND_INT, Text: "42"}},
	})

	problems := gen.CheckDirectives("t", model, spec())
	if len(problems) != 1 || problems[0].Warning {
		t.Fatalf("problems = %v", problems)
	}
	if !strings.Contains(problems[0].Message, "is an integer, want a string") {
		t.Errorf("message = %q", problems[0].Message)
	}
}

// Under-declaring is a plugin bug that should not break a working
// project, so an undeclared directive warns and passes through.
func TestUndeclaredDirectiveWarns(t *testing.T) {
	model := modelWith(&ir.Directive{Name: "surprise", Target: "t"})

	problems := gen.CheckDirectives("t", model, spec())
	if len(problems) != 1 {
		t.Fatalf("problems = %v", problems)
	}
	if !problems[0].Warning {
		t.Error("an undeclared directive was an error")
	}
	if !strings.Contains(problems[0].Message, "surprise") {
		t.Errorf("the warning does not name it: %q", problems[0].Message)
	}
}

// A model carries every target block's directives, so checking one
// backend must not complain about another's.
func TestAnotherTargetsDirectivesAreNotChecked(t *testing.T) {
	model := modelWith(&ir.Directive{Name: "whatever", Target: "other"})
	if problems := gen.CheckDirectives("t", model, spec()); len(problems) != 0 {
		t.Errorf("checked another target's directives: %v", problems)
	}
}

// `out` is tdl's own, read before any backend runs, so a backend is not
// asked to declare it.
func TestCompilerDirectivesAreExempt(t *testing.T) {
	model := &ir.Model{
		Targets: []*ir.TargetBlock{{
			Meta:       &ir.Meta{Name: "t"},
			Directives: []*ir.Directive{{Name: "out", Target: "t", Args: []*ir.Literal{str("./gen")}}},
		}},
	}
	if problems := gen.CheckDirectives("t", model, spec()); len(problems) != 0 {
		t.Errorf("problems = %v", problems)
	}
}

func TestFieldDirectivesAreChecked(t *testing.T) {
	model := &ir.Model{
		Decls: []*ir.Decl{{
			Meta: &ir.Meta{Name: "Order"},
			Node: &ir.Decl_Structure{Structure: &ir.Struct{
				Fields: []*ir.Field{{
					Meta:       &ir.Meta{Name: "id"},
					Directives: []*ir.Directive{{Name: "tag", Target: "t"}},
				}},
			}},
		}},
	}
	if problems := gen.CheckDirectives("t", model, spec()); len(problems) != 1 {
		t.Errorf("a field's directive was not checked: %v", problems)
	}
}
