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
	if len(resp.GetDiagnostics()) != 0 {
		t.Errorf("diagnostics = %+v", resp.GetDiagnostics())
	}
	// Widget is the file's one message, so the directives landed in it.
	contains(t, compile(t, resp),
		"message Widget {",
		`reserved 2, 3; reserved 50, 51; reserved "legacy";`,
		"string id = 1;",
	)
}

// A field a message reserves, by number or by name, is refused: protoc
// rejects the message, so it is skipped and its siblings still generate.
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
			ds := resp.GetDiagnostics()
			if len(ds) != 1 {
				t.Fatalf("diagnostics = %+v, want one warning", ds)
			}
			d := ds[0]
			if d.GetSeverity() != plugin.Severity_SEVERITY_WARNING {
				t.Errorf("severity = %v, want a warning", d.GetSeverity())
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

// Under an edition a reserved name is an identifier, since editions refuse
// the quoted form proto3 uses.
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
			if len(resp.GetDiagnostics()) != 0 {
				t.Errorf("diagnostics = %+v", resp.GetDiagnostics())
			}
			contains(t, compile(t, resp),
				`edition = "`+edition+`";`,
				"reserved 2, 3; reserved 50, 51; reserved legacy;",
			)
		})
	}
}
