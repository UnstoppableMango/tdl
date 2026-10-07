package gen_test

import (
	"path/filepath"
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

// The command line overrides a target block's out directive.
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

// An out directive is relative to the file declaring its target block, the
// way an import or an include is, so where output goes does not depend on
// where tdl runs. An absolute path and -o are taken as given.
func TestOutIsRelativeToTheDeclaringFile(t *testing.T) {
	at := func(b *ir.TargetBlock, file string) *ir.TargetBlock {
		b.Meta.Position = &ir.Position{Filename: file}
		return b
	}
	model := &ir.Model{Targets: []*ir.TargetBlock{
		at(block("go", "../api/model"), filepath.Join("model", "cluster.tdl")),
		at(block("sql", "/abs/sql"), filepath.Join("model", "cluster.tdl")),
	}}

	targets, err := gen.Targets(model, "")
	if err != nil {
		t.Fatalf("targets: %v", err)
	}
	if got, want := targets[0].Out, "api/model"; got != want {
		t.Errorf("relative out = %q, want %q", got, want)
	}
	if got, want := targets[1].Out, "/abs/sql"; got != want {
		t.Errorf("absolute out = %q, want %q", got, want)
	}

	overridden, err := gen.Targets(model, "elsewhere")
	if err != nil {
		t.Fatalf("targets: %v", err)
	}
	if got := overridden[0].Out; got != "elsewhere" {
		t.Errorf("-o went to %q, want it as given", got)
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
