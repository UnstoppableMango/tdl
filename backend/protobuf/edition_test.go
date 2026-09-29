package protobuf_test

import (
	"strconv"
	"testing"

	"github.com/unstoppablemango/tdl/backend/internal/irtest"
	"github.com/unstoppablemango/tdl/backend/protobuf"
	"github.com/unstoppablemango/tdl/internal/gen"
	"github.com/unstoppablemango/tdl/ir"
	"github.com/unstoppablemango/tdl/plugin"
)

// An `edition` directive replaces the proto3 syntax line with an edition
// header. 2023 is the newest edition protocompile compiles.
func TestEditionDirective(t *testing.T) {
	b := irtest.New("shop")
	b.Own(value("Note", irtest.Field("body", b.Named("string"))))
	b.Model.Targets = []*ir.TargetBlock{{
		Meta: &ir.Meta{Name: protobuf.Name},
		Directives: []*ir.Directive{{
			Name: "edition", Target: protobuf.Name, Args: []*ir.Literal{irtest.Text("2023")},
			Position: &ir.Position{Filename: "shop.tdl", Line: 2},
		}},
	}}

	if problems := gen.CheckDirectives(protobuf.Name, b.Model, protobuf.Backend{}.Describe()); len(problems) != 0 {
		t.Errorf("edition should be a declared directive: %+v", problems)
	}

	resp := generate(t, b)
	if len(resp.GetDiagnostics()) != 0 {
		t.Errorf("diagnostics = %+v", resp.GetDiagnostics())
	}
	src := compile(t, resp)
	contains(t, src, `edition = "2023";`, "message Note { string body = 1; }")
	absent(t, src, "syntax")
}

// Under an edition a scalar or enum field has explicit presence by default,
// so `T?` and `T | null` emit no `optional` label, the same as a bare `T`.
func TestEditionOptionalIsBare(t *testing.T) {
	b := irtest.New("shop")
	b.Own(enum("Status", variant("Active")))
	b.Own(value("Ticket",
		irtest.Field("note", b.Named("Option", b.Named("string"))),
		irtest.Field("shipOn", b.Named("Nullable", b.Named("date"))),
		irtest.Field("status", b.Named("Option", b.Named("Status"))),
		irtest.Field("body", b.Named("string")),
	))
	b.Model.Targets = []*ir.TargetBlock{{
		Meta: &ir.Meta{Name: protobuf.Name},
		Directives: []*ir.Directive{{
			Name: "edition", Target: protobuf.Name, Args: []*ir.Literal{irtest.Text("2023")},
			Position: &ir.Position{Filename: "shop.tdl", Line: 2},
		}},
	}}

	resp := generate(t, b)
	if len(resp.GetDiagnostics()) != 0 {
		t.Errorf("diagnostics = %+v", resp.GetDiagnostics())
	}
	src := compile(t, resp)
	contains(t, src,
		`edition = "2023";`,
		"message Ticket { string note = 1; string ship_on = 2; Status status = 3; string body = 4; }",
	)
	absent(t, src, "optional")
}

// The `edition` directive accepts only the editions protobuf defines. Any
// other value is an error at the directive and nothing is written, the same
// as a package protobuf refuses.
func TestEditionValues(t *testing.T) {
	pos := &ir.Position{Filename: "shop.tdl", Line: 2, Column: 3}
	for _, c := range []struct {
		edition string
		header  string
	}{
		{edition: "2023", header: `edition = "2023";`},
		{edition: "2024", header: `edition = "2024";`},
		{edition: ""},
		{edition: "proto3"},
		{edition: "2022"},
		{edition: "banana"},
	} {
		t.Run(strconv.Quote(c.edition), func(t *testing.T) {
			b := irtest.New("shop")
			b.Own(value("Note", irtest.Field("body", b.Named("string"))))
			b.Model.Targets = []*ir.TargetBlock{{
				Meta: &ir.Meta{Name: protobuf.Name},
				Directives: []*ir.Directive{{
					Name: "edition", Target: protobuf.Name, Args: []*ir.Literal{irtest.Text(c.edition)},
					Position: pos,
				}},
			}}
			resp := generate(t, b)

			if c.header != "" {
				if len(resp.GetDiagnostics()) != 0 {
					t.Errorf("diagnostics = %+v", resp.GetDiagnostics())
				}
				if len(resp.GetFiles()) != 1 {
					t.Fatalf("files = %d", len(resp.GetFiles()))
				}
				// protocompile v0.14.1 compiles no edition past 2023, so the
				// header is asserted on the text alone.
				contains(t, string(resp.GetFiles()[0].GetContent()), c.header)
				return
			}

			if len(resp.GetFiles()) != 0 {
				t.Errorf("files = %d, want none", len(resp.GetFiles()))
			}
			diags := resp.GetDiagnostics()
			if len(diags) != 1 {
				t.Fatalf("diagnostics = %+v, want one error", diags)
			}
			if got := diags[0].GetSeverity(); got != plugin.Severity_SEVERITY_ERROR {
				t.Errorf("severity = %v, want error", got)
			}
			if got := diags[0].GetPosition(); got.GetFilename() != pos.GetFilename() ||
				got.GetLine() != pos.GetLine() || got.GetColumn() != pos.GetColumn() {
				t.Errorf("position = %+v, want %+v", got, pos)
			}
		})
	}
}
