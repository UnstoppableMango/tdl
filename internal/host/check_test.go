package host_test

import (
	"slices"
	"strings"
	"testing"

	"google.golang.org/protobuf/proto"

	"github.com/unstoppablemango/tdl/internal/host"
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
	if problems := host.CheckDirectives("t", model, spec()); len(problems) != 0 {
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
			problems := host.CheckDirectives("t", model, spec())
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

	problems := host.CheckDirectives("t", model, spec())
	if len(problems) != 1 || problems[0].GetSeverity() == plugin.Severity_SEVERITY_WARNING {
		t.Fatalf("problems = %v", problems)
	}
	if !strings.Contains(problems[0].GetMessage(), "is an integer, want a string") {
		t.Errorf("message = %q", problems[0].GetMessage())
	}
}

func TestUndeclaredDirectiveWarns(t *testing.T) {
	model := modelWith(&ir.Directive{Name: "surprise", Target: "t"})

	problems := host.CheckDirectives("t", model, spec())
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
	if problems := host.CheckDirectives("t", model, spec()); len(problems) != 0 {
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
	if problems := host.CheckDirectives("t", model, spec()); len(problems) != 0 {
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
	if problems := host.CheckDirectives("t", model, spec()); len(problems) != 1 {
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

	problems := host.CheckDirectives("t", model, spec())
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
	for _, p := range host.CheckDirectives("t", model, placedSpec()) {
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
		for _, p := range host.CheckDirectives("t", model, placedSpec()) {
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
	problems := host.CheckDirectives("t", model, placedSpec())
	if len(problems) != 1 || !strings.Contains(problems[0].GetMessage(), "two entries") {
		t.Errorf("problems = %v", problems)
	}
}

// Every target accepts `post` on its block with one or more string
// arguments, whether or not its backend declares it.
func TestPostIsCheckedForEveryTarget(t *testing.T) {
	post := func(args ...*ir.Literal) *ir.Directive {
		return &ir.Directive{Name: host.PostDirective, Target: "t", Args: args}
	}
	block := func(ds ...*ir.Directive) *ir.Model {
		return &ir.Model{Targets: []*ir.TargetBlock{{Meta: &ir.Meta{Name: "t"}, Directives: ds}}}
	}
	num := &ir.Literal{Kind: ir.LiteralKind_LITERAL_KIND_INT, Text: "1"}

	tests := []struct {
		name  string
		model *ir.Model
		want  string
	}{
		{"one", block(post(str("biome"))), ""},
		{"several", block(post(str("biome"), str("sed"))), ""},
		{"no arguments", block(post()), "post takes at least 1 argument(s), got 0"},
		{"a later argument not a string", block(post(str("biome"), num)), "post argument 2 is an integer, want a string"},
		{"on a declaration", modelWith(post(str("biome"))), "post is written on a declaration, and t reads it only on the target block"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			problems := host.CheckDirectives("t", tc.model, spec())
			if tc.want == "" {
				if len(problems) != 0 {
					t.Errorf("problems = %v", problems)
				}
				return
			}
			if len(problems) != 1 || !strings.Contains(problems[0].GetMessage(), tc.want) || !host.Fatal(problems) {
				t.Errorf("problems = %v, want one error containing %q", problems, tc.want)
			}
		})
	}
}

// A bare roundtrip on the block is the deprecated spelling of annotate: it
// warns once and is checked as annotate, while roundtrip with arguments or
// on a declaration is a directive of its own.
func TestRoundtripIsDeprecatedForAnnotate(t *testing.T) {
	desc := plugin.Description{Name: "t", Directives: []*plugin.DirectiveSpec{
		{Name: "annotate", Placements: []plugin.Placement{plugin.Placement_PLACEMENT_BLOCK}},
	}}
	block := &ir.Model{Targets: []*ir.TargetBlock{{
		Meta:       &ir.Meta{Name: "t"},
		Directives: []*ir.Directive{{Name: "roundtrip", Target: "t"}},
	}}}
	problems := host.CheckDirectives("t", block, desc)
	if len(problems) != 1 || problems[0].GetSeverity() != plugin.Severity_SEVERITY_WARNING ||
		problems[0].GetMessage() != "roundtrip is deprecated; write annotate" {
		t.Errorf("problems = %v", problems)
	}

	if got := host.Rename(&ir.Directive{Name: "roundtrip", Args: []*ir.Literal{str("x")}}, desc); got != "roundtrip" {
		t.Errorf("roundtrip with an argument is read as %s", got)
	}
	both := proto.CloneOf(block)
	both.Targets[0].Directives = append(both.Targets[0].Directives, &ir.Directive{Name: "annotate", Target: "t"})
	if problems := host.CheckDirectives("t", both, desc); len(problems) != 2 || !strings.Contains(problems[1].GetMessage(), "set annotate") {
		t.Errorf("roundtrip beside annotate = %v", problems)
	}

	// A backend still declaring roundtrip, or not declaring annotate,
	// reads roundtrip as itself.
	old := plugin.Description{Name: "t", Directives: []*plugin.DirectiveSpec{{Name: "roundtrip"}, {Name: "annotate"}}}
	if problems := host.CheckDirectives("t", block, old); len(problems) != 0 {
		t.Errorf("problems where roundtrip is declared = %v", problems)
	}
	problems = host.CheckDirectives("t", block, plugin.Description{Name: "t"})
	if len(problems) != 1 || problems[0].GetMessage() != "t does not declare the directive roundtrip" {
		t.Errorf("problems where annotate is not declared = %v", problems)
	}
	problems = host.CheckDirectives("t", modelWith(&ir.Directive{Name: "roundtrip", Target: "t"}), desc)
	if len(problems) != 1 || problems[0].GetMessage() != "t does not declare the directive roundtrip" {
		t.Errorf("problems on a declaration = %v", problems)
	}
}
