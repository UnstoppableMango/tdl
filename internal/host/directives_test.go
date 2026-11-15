package host_test

import (
	"strings"
	"testing"

	"github.com/unstoppablemango/tdl/internal/host"
	"github.com/unstoppablemango/tdl/ir"
	"github.com/unstoppablemango/tdl/plugin"
)

func blockWith(directives ...*ir.Directive) *ir.Model {
	return &ir.Model{Targets: []*ir.TargetBlock{{
		Meta:       &ir.Meta{Name: "t"},
		Directives: directives,
	}}}
}

// Every backend accepts the host's directives in its target block, whether
// or not it declares them, and more than one of each.
func TestHostDirectivesFit(t *testing.T) {
	for _, name := range host.Directives() {
		model := blockWith(
			&ir.Directive{Name: name, Target: "t", Args: []*ir.Literal{str("a"), str("b")}},
			&ir.Directive{Name: name, Target: "t", Args: []*ir.Literal{str("c")}},
		)
		if problems := host.CheckDirectives("t", model, spec()); len(problems) != 0 {
			t.Errorf("%s: problems = %v", name, problems)
		}
	}
}

// A host directive is held to the host's spec: names as strings, at least
// one, in the target block only, whatever the backend declares.
func TestHostDirectiveMisuse(t *testing.T) {
	num := &ir.Literal{Kind: ir.LiteralKind_LITERAL_KIND_INT, Text: "1"}
	for _, name := range host.Directives() {
		tests := []struct {
			desc  string
			model *ir.Model
			want  string
		}{
			{"no arguments", blockWith(&ir.Directive{Name: name, Target: "t"}), "at least 1"},
			{"a number", blockWith(&ir.Directive{Name: name, Target: "t", Args: []*ir.Literal{str("a"), num}}), "argument 2 is"},
			{"on a declaration", modelWith(&ir.Directive{Name: name, Target: "t", Args: []*ir.Literal{str("a")}}), "the target block"},
		}
		for _, tt := range tests {
			t.Run(name+"/"+tt.desc, func(t *testing.T) {
				desc := spec()
				desc.Directives = append(desc.Directives, &plugin.DirectiveSpec{Name: name, MaxArgs: -1})
				problems := host.CheckDirectives("t", tt.model, desc)
				if len(problems) != 1 || problems[0].GetSeverity() != plugin.Severity_SEVERITY_ERROR {
					t.Fatalf("problems = %v", problems)
				}
				if !strings.Contains(problems[0].GetMessage(), tt.want) {
					t.Errorf("message = %q, want it to mention %q", problems[0].GetMessage(), tt.want)
				}
			})
		}
	}
}

// Names reads every name a target block's directive gives, in order.
func TestNames(t *testing.T) {
	block := &ir.TargetBlock{Meta: &ir.Meta{Name: "t"}, Directives: []*ir.Directive{
		{Name: host.PostDirective, Args: []*ir.Literal{str("a"), str("b")}},
		{Name: "out", Args: []*ir.Literal{str("gen")}},
		{Name: host.PostDirective, Args: []*ir.Literal{str("c")}},
	}}
	got := host.Names(block, host.PostDirective)
	if strings.Join(got, ",") != "a,b,c" {
		t.Errorf("names = %v", got)
	}
}
