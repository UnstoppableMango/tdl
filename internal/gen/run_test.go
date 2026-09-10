package gen_test

import (
	"testing"

	"github.com/unstoppablemango/tdl/internal/gen"
	"github.com/unstoppablemango/tdl/ir"
)

func block(name, out string) *ir.TargetBlock {
	b := &ir.TargetBlock{Meta: &ir.Meta{Name: name}}
	if out != "" {
		b.Directives = []*ir.Directive{{
			Name:   "out",
			Target: name,
			Args:   []*ir.Literal{{Kind: ir.LiteralKind_LITERAL_KIND_STRING, Text: out}},
		}}
	}
	return b
}

// A target block declares where its output goes; the command line
// overrides it for one invocation.
func TestTargetsReadOutDirective(t *testing.T) {
	model := &ir.Model{Targets: []*ir.TargetBlock{block("go", "./gen/go"), block("sql", "./gen/sql")}}

	targets, err := gen.Targets(model, "")
	if err != nil {
		t.Fatalf("targets: %v", err)
	}
	if len(targets) != 2 || targets[0].Out != "./gen/go" || targets[1].Out != "./gen/sql" {
		t.Fatalf("targets = %+v", targets)
	}

	overridden, err := gen.Targets(model, "/tmp/elsewhere")
	if err != nil {
		t.Fatalf("targets: %v", err)
	}
	for _, target := range overridden {
		if target.Out != "/tmp/elsewhere" {
			t.Errorf("%s went to %q", target.Name, target.Out)
		}
	}
}

func TestTargetWithoutOut(t *testing.T) {
	model := &ir.Model{Targets: []*ir.TargetBlock{block("go", "")}}
	if _, err := gen.Targets(model, ""); err == nil {
		t.Fatal("a target with no out and no -o was accepted")
	}
}

func TestBuiltinRegistry(t *testing.T) {
	if _, ok := gen.Builtin("debug"); !ok {
		t.Error("debug is not registered")
	}
	if _, ok := gen.Builtin("nonesuch"); ok {
		t.Error("an unknown name resolved to a backend")
	}
	if names := gen.BuiltinNames(); len(names) == 0 {
		t.Error("no backends are compiled in")
	}
}
