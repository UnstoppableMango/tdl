package sema

import (
	"strings"
	"testing"

	"github.com/unstoppablemango/tdl/ir"
	"github.com/unstoppablemango/tdl/parser"
)

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

func TestDependencyDeclDirectivesReachTheExtern(t *testing.T) {
	deps := MapLoader{
		"dep/cli.tdl": `package acme.cli.v1

type Utility { name: string }
type Tool { name: string }
type Kit { name: string }

target protobuf for acme.cli.v1 {
  Utility => file("cli.proto")
  Utility.name => number(4)
  Tool {
    file("tool.proto")
    name("Instrument")
  }
  Kit => file("kit.proto")
}
`,
	}

	tests := []struct {
		name, src string
		want      map[string][]string // extern to its directives, as name(arg)
	}{
		{
			name: "alias",
			src: `package acme.ops.v1

import "dep/cli.tdl" as cli

type Job { utility: cli.Utility  tool: cli.Tool }
`,
			want: map[string][]string{
				"Utility": {`file("cli.proto")`},
				"Tool":    {`file("tool.proto")`, `name("Instrument")`},
			},
		},
		{
			name: "the root's entry wins",
			src: `package acme.ops.v1

import "dep/cli.tdl" as _

type Job { utility: Utility  kit: Kit }

target protobuf for acme.ops.v1 {
  Kit => file("mine.proto")
}
`,
			want: map[string][]string{
				"Utility": {`file("cli.proto")`},
				"Tool":    {`file("tool.proto")`, `name("Instrument")`},
				"Kit":     {`file("mine.proto")`},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file, err := parser.Parse("main.tdl", strings.NewReader(tt.src))
			if err != nil {
				t.Fatalf("unexpected parse error: %v", err)
			}
			model, diags := Lower(file, WithLoader(deps))
			if len(diags) > 0 {
				t.Fatalf("unexpected diagnostics: %v", diags)
			}

			got := map[string][]string{}
			for _, ext := range model.GetExterns() {
				for _, d := range ext.GetDirectives() {
					got[ext.GetName()] = append(got[ext.GetName()], d.GetName()+`("`+d.GetArgs()[0].GetText()+`")`)
				}
			}
			if len(got) != len(tt.want) {
				t.Fatalf("extern directives = %v, want %v", got, tt.want)
			}
			for name, want := range tt.want {
				if strings.Join(got[name], " ") != strings.Join(want, " ") {
					t.Errorf("%s directives = %v, want %v", name, got[name], want)
				}
			}
		})
	}
}
