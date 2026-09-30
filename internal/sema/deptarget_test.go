package sema

import (
	"strings"
	"testing"

	"github.com/unstoppablemango/tdl/ir"
	"github.com/unstoppablemango/tdl/parser"
)

// A dependency's block-scope directives reach the import that brought it
// in, so a backend generating a reference into the dependency knows which
// file and package the dependency says its declarations live in.
func TestDependencyBlockDirectivesReachTheImport(t *testing.T) {
	file, err := parser.Parse("main.tdl", strings.NewReader(`
package acme.billing.v1

import "dep/money.tdl" as money

type Invoice { total: money.Money }
`))
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}

	model, diags := Lower(file, WithLoader(MapLoader{
		"dep/money.tdl": `package acme.money.v1

type Money { units: int }

target protobuf for acme.money.v1 {
  file("money.proto")
  package("acme.money.v1")
}
`,
	}))
	if len(diags) > 0 {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if got := len(model.GetImports()); got != 1 {
		t.Fatalf("got %d imports, want 1", got)
	}

	type want struct{ target, name, arg string }
	var got []want
	for _, d := range model.GetImports()[0].GetDirectives() {
		w := want{target: d.GetTarget(), name: d.GetName()}
		if args := d.GetArgs(); len(args) == 1 && args[0].GetKind() == ir.LiteralKind_LITERAL_KIND_STRING {
			w.arg = args[0].GetText()
		}
		got = append(got, w)
	}

	expected := []want{
		{target: "protobuf", name: "file", arg: "money.proto"},
		{target: "protobuf", name: "package", arg: "acme.money.v1"},
	}
	if len(got) != len(expected) {
		t.Fatalf("import directives = %+v, want %+v", got, expected)
	}
	for i := range expected {
		if got[i] != expected[i] {
			t.Errorf("import directive %d = %+v, want %+v", i, got[i], expected[i])
		}
	}
}
