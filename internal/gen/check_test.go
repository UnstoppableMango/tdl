package gen_test

import (
	"bytes"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/unstoppablemango/tdl/internal/gen"
	"github.com/unstoppablemango/tdl/internal/sema"
	"github.com/unstoppablemango/tdl/ir"
	"github.com/unstoppablemango/tdl/parser"
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
			if len(problems) != 1 || problems[0].GetSeverity() == plugin.Severity_SEVERITY_WARNING {
				t.Fatalf("problems = %v", problems)
			}
			if !strings.Contains(problems[0].GetMessage(), tt.want) {
				t.Errorf("message = %q, want it to mention %q", problems[0].GetMessage(), tt.want)
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
	if len(problems) != 1 || problems[0].GetSeverity() == plugin.Severity_SEVERITY_WARNING {
		t.Fatalf("problems = %v", problems)
	}
	if !strings.Contains(problems[0].GetMessage(), "is an integer, want a string") {
		t.Errorf("message = %q", problems[0].GetMessage())
	}
}

func TestUndeclaredDirectiveWarns(t *testing.T) {
	model := modelWith(&ir.Directive{Name: "surprise", Target: "t"})

	problems := gen.CheckDirectives("t", model, spec())
	if len(problems) != 1 {
		t.Fatalf("problems = %v", problems)
	}
	if problems[0].GetSeverity() != plugin.Severity_SEVERITY_WARNING {
		t.Error("an undeclared directive was an error")
	}
	if !strings.Contains(problems[0].GetMessage(), "surprise") {
		t.Errorf("the warning does not name it: %q", problems[0].GetMessage())
	}
}

func TestAnotherTargetsDirectivesAreNotChecked(t *testing.T) {
	model := modelWith(&ir.Directive{Name: "whatever", Target: "other"})
	if problems := gen.CheckDirectives("t", model, spec()); len(problems) != 0 {
		t.Errorf("checked another target's directives: %v", problems)
	}
}

// A backend need not declare `out`.
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

func TestVariantDirectivesAreChecked(t *testing.T) {
	model := &ir.Model{Decls: []*ir.Decl{{
		Meta: &ir.Meta{Name: "Payment"},
		Node: &ir.Decl_Enumeration{Enumeration: &ir.Enum{Variants: []*ir.Variant{{
			Meta:       &ir.Meta{Name: "Card"},
			Directives: []*ir.Directive{{Name: "tag", Target: "t"}},
			Fields: []*ir.Field{{
				Meta:       &ir.Meta{Name: "last4"},
				Directives: []*ir.Directive{{Name: "slice", Target: "t", Args: []*ir.Literal{str("x"), str("y")}}},
			}},
		}}}},
	}}}

	problems := gen.CheckDirectives("t", model, spec())
	if len(problems) != 2 {
		t.Fatalf("problems = %v", problems)
	}
	if !strings.Contains(problems[0].GetMessage(), "at least 1") {
		t.Errorf("variant directive: %v", problems[0])
	}
	if !strings.Contains(problems[1].GetMessage(), "at most 0") {
		t.Errorf("variant field directive: %v", problems[1])
	}
}

// placedSpec places `top` on the block, `col` on fields, and `own` on
// declarations, and limits `mode`'s value.
func placedSpec() plugin.Description {
	return plugin.Description{
		Name: "t",
		Directives: []*plugin.DirectiveSpec{
			{Name: "top", Placements: plugin.On(plugin.OnBlock)},
			{Name: "col", Placements: plugin.On(plugin.OnField)},
			{Name: "own", Placements: plugin.On(plugin.OnDecl)},
			{Name: "mode", MinArgs: 1, MaxArgs: 1, Values: []string{"fast", "slow"}},
		},
	}
}

func TestDirectivePlacement(t *testing.T) {
	model := &ir.Model{
		Targets: []*ir.TargetBlock{{
			Meta:       &ir.Meta{Name: "t"},
			Directives: []*ir.Directive{{Name: "top", Target: "t"}, {Name: "col", Target: "t"}},
		}},
		Decls: []*ir.Decl{{
			Meta:       &ir.Meta{Name: "Order"},
			Directives: []*ir.Directive{{Name: "own", Target: "t"}, {Name: "top", Target: "t"}},
			Node: &ir.Decl_Structure{Structure: &ir.Struct{Fields: []*ir.Field{{
				Meta:       &ir.Meta{Name: "id"},
				Directives: []*ir.Directive{{Name: "col", Target: "t"}},
			}}}},
		}},
		// An extern is a declaration, so own is accepted on it.
		Externs: []*ir.Extern{{Name: "other.Thing", Directives: []*ir.Directive{{Name: "own", Target: "t"}}}},
	}

	var got []string
	for _, p := range gen.CheckDirectives("t", model, placedSpec()) {
		got = append(got, p.GetMessage())
	}
	want := []string{
		"col is written on the target block, and t reads it only on a field",
		"top is written on a declaration, and t reads it only on the target block",
	}
	if !slices.Equal(got, want) {
		t.Errorf("problems = %q, want %q", got, want)
	}
}

func TestDirectiveValues(t *testing.T) {
	for value, want := range map[string]string{
		"fast": "",
		"warp": `mode does not take "warp"; use "fast" or "slow"`,
	} {
		model := modelWith(&ir.Directive{Name: "mode", Target: "t", Args: []*ir.Literal{str(value)}})
		var got string
		for _, p := range gen.CheckDirectives("t", model, placedSpec()) {
			got = p.GetMessage()
		}
		if got != want {
			t.Errorf("mode(%q): problem %q, want %q", value, got, want)
		}
	}
}

func TestBlockRepeatsAreTies(t *testing.T) {
	model := &ir.Model{
		Targets: []*ir.TargetBlock{{
			Meta:       &ir.Meta{Name: "t"},
			Directives: []*ir.Directive{{Name: "top", Target: "t"}, {Name: "top", Target: "t"}},
		}},
	}
	problems := gen.CheckDirectives("t", model, placedSpec())
	if len(problems) != 1 || !strings.Contains(problems[0].GetMessage(), "two entries") {
		t.Errorf("problems = %v", problems)
	}
}

// TestCorporaPlaceDirectives holds every built-in backend's placements to
// what the files in the repository write.
func TestCorporaPlaceDirectives(t *testing.T) {
	for path, model := range corpora(t) {
		for _, block := range model.GetTargets() {
			name := block.GetMeta().GetName()
			backend, ok := gen.Builtin(name)
			if !ok {
				continue
			}
			for _, p := range gen.CheckDirectives(name, model, backend.Describe()) {
				if p.GetSeverity() == plugin.Severity_SEVERITY_ERROR {
					t.Errorf("%s: %s: %s", path, name, p.GetMessage())
				}
			}
		}
	}
}

// corpora lowers every file in the corpora and examples that lowers, by
// path.
func corpora(t *testing.T) map[string]*ir.Model {
	t.Helper()
	var paths []string
	for _, glob := range []string{
		"../../testdata/conformance/*/source.tdl",
		"../../testdata/gen/*/source.tdl",
		"../../testdata/roundtrip/*/*/*.tdl",
		"../../examples/*.tdl",
	} {
		found, err := filepath.Glob(glob)
		if err != nil {
			t.Fatal(err)
		}
		paths = append(paths, found...)
	}

	models := map[string]*ir.Model{}
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		file, err := parser.Parse(path, bytes.NewReader(data))
		if err != nil {
			continue // the invalid corpus and pending cases are checked elsewhere
		}
		model, diags := sema.Lower(file, sema.WithLoader(sema.FSLoader{}))
		if len(diags) > 0 {
			continue
		}
		models[path] = model
	}
	return models
}
