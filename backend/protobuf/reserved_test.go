package protobuf_test

import (
	"context"
	"strings"
	"testing"

	"github.com/unstoppablemango/tdl/backend/protobuf"
	"github.com/unstoppablemango/tdl/internal/gen"
	"github.com/unstoppablemango/tdl/internal/sema"
	"github.com/unstoppablemango/tdl/parser"
	"github.com/unstoppablemango/tdl/plugin"
)

const reservedModel = `package shop

type Widget { id: string }

target protobuf for shop {
  Widget {
    reserved(2, 3)
    reserved(50, 51)
    reserved("legacy")
  }
}
`

func TestReservedOnAMessage(t *testing.T) {
	file, err := parser.Parse("shop.tdl", strings.NewReader(reservedModel))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	model, diags := sema.Lower(file)
	if len(diags) > 0 {
		t.Fatalf("lowering reported %v", diags)
	}

	if problems := gen.CheckDirectives(protobuf.Name, model, protobuf.Backend{}.Describe()); len(problems) != 0 {
		t.Errorf("problems = %v, want none: protobuf declares reserved, and repeatable", problems)
	}

	resp, err := protobuf.Backend{}.Generate(context.Background(), &plugin.Request{Target: protobuf.Name, Model: model})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if len(uncoded(resp.GetDiagnostics())) != 0 {
		t.Errorf("diagnostics = %+v", uncoded(resp.GetDiagnostics()))
	}
	// Widget is the file's one message, so the directives landed in it.
	contains(t, compile(t, resp),
		"message Widget {",
		`reserved 2, 3; reserved 50, 51; reserved "legacy";`,
		"string id = 1;",
	)
}

// A field on a reserved number or name is an error, which skips its message;
// siblings still generate.
func TestReservedFieldIsRefused(t *testing.T) {
	for _, tt := range []struct {
		name  string
		model string
		line  int32
		wants []string
	}{
		{
			name: "a reserved number",
			model: `package shop

type Widget {
  id: string
  name: string
}

type Gadget { id: string }

target protobuf for shop {
  Widget {
    reserved(2)
    id => number(1)
    name => number(2)
  }
}
`,
			// The pinned number's directive.
			line:  14,
			wants: []string{"Widget.name", "2", "reserve"},
		},
		{
			name: "a reserved name",
			model: `package shop

type Widget {
  id: string
  legacy: string
}

type Gadget { id: string }

target protobuf for shop {
  Widget {
    reserved("legacy")
  }
}
`,
			// The field's declaration.
			line:  5,
			wants: []string{"Widget.legacy", "reserve"},
		},
		{
			name: "an inlined oneof member pinned to a reserved number",
			model: `package shop

enum Actor {
  Contact { contact: string }
  System { system: string }
}

type Widget {
  id: string
  actor: Actor
}

type Gadget { id: string }

target protobuf for shop {
  Actor {
    System => number(3)
  }
  Widget {
    reserved(3)
    actor => oneof
  }
}
`,
			// The pinned number's directive.
			line:  17,
			wants: []string{"Widget.System", "3", "reserve"},
		},
		{
			name: "an inlined oneof member's name",
			model: `package shop

enum Actor {
  Contact { contact: string }
  System { system: string }
}

type Widget {
  id: string
  actor: Actor
}

type Gadget { id: string }

target protobuf for shop {
  Widget {
    reserved("system")
    actor => oneof
  }
}
`,
			line:  5,
			wants: []string{"Widget.System", "system", "reserve"},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			file, err := parser.Parse("shop.tdl", strings.NewReader(tt.model))
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			model, diags := sema.Lower(file)
			if len(diags) > 0 {
				t.Fatalf("lowering reported %v", diags)
			}

			resp, err := protobuf.Backend{}.Generate(context.Background(), &plugin.Request{Target: protobuf.Name, Model: model})
			if err != nil {
				t.Fatalf("generate: %v", err)
			}
			ds := uncoded(resp.GetDiagnostics())
			if len(ds) != 1 {
				t.Fatalf("diagnostics = %+v, want one error", ds)
			}
			d := ds[0]
			if d.GetSeverity() != plugin.Severity_SEVERITY_ERROR {
				t.Errorf("severity = %v, want an error", d.GetSeverity())
			}
			if d.GetPosition().GetFilename() != "shop.tdl" || d.GetPosition().GetLine() != tt.line {
				t.Errorf("position = %v, want shop.tdl:%d", d.GetPosition(), tt.line)
			}
			for _, w := range tt.wants {
				if !strings.Contains(d.GetMessage(), w) {
					t.Errorf("message %q does not mention %q", d.GetMessage(), w)
				}
			}

			src := compile(t, resp)
			absent(t, src, "message Widget {")
			contains(t, src, "message Gadget {")
		})
	}
}

// Under an edition a reserved name is unquoted.
func TestReservedNameUnderAnEdition(t *testing.T) {
	for _, edition := range []string{"2023", "2024"} {
		t.Run(edition, func(t *testing.T) {
			model := strings.Replace(reservedModel, "for shop {\n", "for shop {\n  edition(\""+edition+"\")\n", 1)
			file, err := parser.Parse("shop.tdl", strings.NewReader(model))
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			lowered, diags := sema.Lower(file)
			if len(diags) > 0 {
				t.Fatalf("lowering reported %v", diags)
			}

			resp, err := protobuf.Backend{}.Generate(context.Background(), &plugin.Request{Target: protobuf.Name, Model: lowered})
			if err != nil {
				t.Fatalf("generate: %v", err)
			}
			if len(uncoded(resp.GetDiagnostics())) != 0 {
				t.Errorf("diagnostics = %+v", uncoded(resp.GetDiagnostics()))
			}
			contains(t, compile(t, resp),
				`edition = "`+edition+`";`,
				"reserved 2, 3; reserved 50, 51; reserved legacy;",
			)
		})
	}
}

func TestUnpinnedFieldsSkipReservedNumbers(t *testing.T) {
	const model = `package shop

type Widget {
  a: string
  b: string
  c: string
}

target protobuf for shop {
  Widget {
    reserved(2, 3)
  }
}
`
	file, err := parser.Parse("shop.tdl", strings.NewReader(model))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	lowered, diags := sema.Lower(file)
	if len(diags) > 0 {
		t.Fatalf("lowering reported %v", diags)
	}

	resp, err := protobuf.Backend{}.Generate(context.Background(), &plugin.Request{Target: protobuf.Name, Model: lowered})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if len(uncoded(resp.GetDiagnostics())) != 0 {
		t.Errorf("diagnostics = %+v, want none", uncoded(resp.GetDiagnostics()))
	}
	contains(t, compile(t, resp),
		"message Widget {",
		"reserved 2, 3;",
		"string a = 1; string b = 4; string c = 5;",
	)
}

func TestUnpinnedOneofMembersSkipReservedNumbers(t *testing.T) {
	const model = `package shop

enum Actor {
  Contact { contact: string }
  System { system: string }
}

type Widget {
  id: string
  actor: Actor
}

target protobuf for shop {
  Widget {
    reserved(2)
    actor => oneof
  }
}
`
	file, err := parser.Parse("shop.tdl", strings.NewReader(model))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	lowered, diags := sema.Lower(file)
	if len(diags) > 0 {
		t.Fatalf("lowering reported %v", diags)
	}

	resp, err := protobuf.Backend{}.Generate(context.Background(), &plugin.Request{Target: protobuf.Name, Model: lowered})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if len(uncoded(resp.GetDiagnostics())) != 0 {
		t.Errorf("diagnostics = %+v, want none", uncoded(resp.GetDiagnostics()))
	}
	contains(t, compile(t, resp),
		"message Widget {",
		"reserved 2;",
		"string id = 1; oneof actor { string contact = 3; string system = 4; }",
	)
}

func TestPinnedFieldOnAReservedNumberIsRefused(t *testing.T) {
	const model = `package shop

type Widget {
  a: string
  b: string
  c: string
}

type Gadget { id: string }

target protobuf for shop {
  Widget {
    reserved(2, 3)
    b => number(2)
  }
}
`
	file, err := parser.Parse("shop.tdl", strings.NewReader(model))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	lowered, diags := sema.Lower(file)
	if len(diags) > 0 {
		t.Fatalf("lowering reported %v", diags)
	}

	resp, err := protobuf.Backend{}.Generate(context.Background(), &plugin.Request{Target: protobuf.Name, Model: lowered})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	ds := uncoded(resp.GetDiagnostics())
	if len(ds) != 1 {
		t.Fatalf("diagnostics = %+v, want one error", ds)
	}
	d := ds[0]
	if d.GetSeverity() != plugin.Severity_SEVERITY_ERROR {
		t.Errorf("severity = %v, want an error", d.GetSeverity())
	}
	// The pin's directive.
	if d.GetPosition().GetFilename() != "shop.tdl" || d.GetPosition().GetLine() != 14 {
		t.Errorf("position = %v, want shop.tdl:14", d.GetPosition())
	}
	for _, w := range []string{"Widget.b", "2", "reserve"} {
		if !strings.Contains(d.GetMessage(), w) {
			t.Errorf("message %q does not mention %q", d.GetMessage(), w)
		}
	}

	src := compile(t, resp)
	absent(t, src, "message Widget {")
	contains(t, src, "message Gadget {")
}
