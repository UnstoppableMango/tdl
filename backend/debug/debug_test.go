package debug_test

import (
	"context"
	"strings"
	"testing"

	"github.com/unstoppablemango/tdl/backend/debug"
	"github.com/unstoppablemango/tdl/ir"
	"github.com/unstoppablemango/tdl/plugin"
)

func TestDescribe(t *testing.T) {
	d := debug.Backend{}.Describe()
	if d.Name != debug.Name {
		t.Errorf("name = %q", d.Name)
	}
	if len(d.Directives) != 1 || d.Directives[0].GetName() != "note" {
		t.Errorf("directives = %+v", d.Directives)
	}
}

func TestGenerate(t *testing.T) {
	model := &ir.Model{
		Package: "shop",
		Decls: []*ir.Decl{
			{
				Meta: &ir.Meta{
					Name:     "Order",
					Position: &ir.Position{Filename: "shop.tdl"},
				},
				Directives: []*ir.Directive{
					{Name: "note", Target: "debug"},
					{Name: "table", Target: "sql"},
				},
				Node: &ir.Decl_Structure{Structure: &ir.Struct{
					Fields: []*ir.Field{{
						Meta: &ir.Meta{Name: "id"},
						Type: &ir.ID{Name: "string"},
					}},
				}},
			},
			{
				Meta: &ir.Meta{
					Name:     "List",
					Position: &ir.Position{Filename: "/nix/store/x/std.tdl"},
				},
			},
		},
	}

	resp, err := debug.Backend{}.Generate(context.Background(), &plugin.Request{
		Target: "debug",
		Model:  model,
	})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if len(resp.GetFiles()) != 1 || resp.GetFiles()[0].GetPath() != "model.txt" {
		t.Fatalf("files = %+v", resp.GetFiles())
	}

	out := string(resp.GetFiles()[0].GetContent())
	for _, want := range []string{
		"package shop",
		"target debug",
		"1 own, 1 from the prelude",
		"Order",
		"field id: string",
		"directive note",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}

	// A model carries directives for every target block, so a backend that
	// does not filter would report another backend's.
	if strings.Contains(out, "directive table") {
		t.Errorf("another target's directive leaked through:\n%s", out)
	}
}

// A backend says what it cannot handle in a diagnostic rather than by
// returning an error, because a diagnostic reaches the user with a
// position attached and does not stop the run.
func TestDiagnostics(t *testing.T) {
	model := &ir.Model{
		Package: "shop",
		Decls: []*ir.Decl{
			{
				Meta: &ir.Meta{Name: "Empty", Position: &ir.Position{Filename: "shop.tdl", Line: 3}},
				Node: &ir.Decl_Structure{Structure: &ir.Struct{}},
			},
			{
				Meta: &ir.Meta{Name: "Full", Position: &ir.Position{Filename: "shop.tdl", Line: 7}},
				Node: &ir.Decl_Structure{Structure: &ir.Struct{
					Fields: []*ir.Field{{Meta: &ir.Meta{Name: "id"}, Type: &ir.ID{Name: "string"}}},
				}},
			},
		},
	}

	resp, err := debug.Backend{}.Generate(context.Background(), &plugin.Request{Target: "debug", Model: model})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if len(resp.GetDiagnostics()) != 1 {
		t.Fatalf("diagnostics = %+v", resp.GetDiagnostics())
	}

	d := resp.GetDiagnostics()[0]
	if d.GetSeverity() != plugin.Severity_SEVERITY_WARNING {
		t.Errorf("severity = %v", d.GetSeverity())
	}
	if d.GetPosition().GetLine() != 3 {
		t.Errorf("position = %+v", d.GetPosition())
	}

	// A warning does not stop a run, so the file is still there.
	if len(resp.GetFiles()) != 1 {
		t.Error("a warning suppressed the output")
	}
}
