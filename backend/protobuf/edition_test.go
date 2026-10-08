package protobuf_test

import (
	"strconv"
	"strings"
	"testing"

	"github.com/unstoppablemango/tdl/backend/internal/irtest"
	"github.com/unstoppablemango/tdl/backend/protobuf"
	"github.com/unstoppablemango/tdl/internal/gen"
	"github.com/unstoppablemango/tdl/ir"
	"github.com/unstoppablemango/tdl/plugin"
)

// The edition header replaces the proto3 syntax line.
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
	if len(uncoded(resp.GetDiagnostics())) != 0 {
		t.Errorf("diagnostics = %+v", uncoded(resp.GetDiagnostics()))
	}
	src := compile(t, resp)
	contains(t, src, `edition = "2023";`, "message Note { string body = 1; }")
	absent(t, src, "syntax")
}

// Under an edition `T?` and `T | null` emit no `optional` label, since
// every field has explicit presence.
func TestEditionOptionalIsBare(t *testing.T) {
	for _, edition := range []string{"2023", "2024"} {
		t.Run(edition, func(t *testing.T) {
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
					Name: "edition", Target: protobuf.Name, Args: []*ir.Literal{irtest.Text(edition)},
					Position: &ir.Position{Filename: "shop.tdl", Line: 2},
				}},
			}}

			resp := generate(t, b)
			if len(uncoded(resp.GetDiagnostics())) != 0 {
				t.Errorf("diagnostics = %+v", uncoded(resp.GetDiagnostics()))
			}
			src := compile(t, resp)
			contains(t, src,
				`edition = "`+edition+`";`,
				"message Ticket { string note = 1; string ship_on = 2; Status status = 3; string body = 4; }",
			)
			absent(t, src, "optional")
		})
	}
}

// An edition protobuf does not define is an error at the directive, and
// nothing is written.
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
				if len(uncoded(resp.GetDiagnostics())) != 0 {
					t.Errorf("diagnostics = %+v", uncoded(resp.GetDiagnostics()))
				}
				contains(t, compile(t, resp), c.header)
				return
			}

			if len(resp.GetFiles()) != 0 {
				t.Errorf("files = %d, want none", len(resp.GetFiles()))
			}
			diags := uncoded(resp.GetDiagnostics())
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

// Every file a `file` directive splits out declares the edition.
func TestEditionInEveryFile(t *testing.T) {
	b := irtest.New("acme.cli.v1")
	token := value("Token", irtest.Field("text", b.Named("string")))
	token.Directives = []*ir.Directive{{
		Name: "file", Target: protobuf.Name, Args: []*ir.Literal{irtest.Text("cst.proto")},
		Position: &ir.Position{Filename: "cli.tdl", Line: 5},
	}}
	b.Own(token)
	b.Own(value("Command", irtest.Field("tokens", b.Named("List", b.Named("Token")))))
	b.Model.Targets = []*ir.TargetBlock{{
		Meta: &ir.Meta{Name: protobuf.Name},
		Directives: []*ir.Directive{{
			Name: "edition", Target: protobuf.Name, Args: []*ir.Literal{irtest.Text("2024")},
			Position: &ir.Position{Filename: "cli.tdl", Line: 2},
		}},
	}}

	resp := generate(t, b)
	if len(uncoded(resp.GetDiagnostics())) != 0 {
		t.Errorf("diagnostics = %+v", uncoded(resp.GetDiagnostics()))
	}
	files := compileAll(t, resp)
	if len(files) != 2 {
		t.Fatalf("files = %d, want 2: %v", len(files), files)
	}
	for path, src := range files {
		if !strings.Contains(src, `edition = "2024";`) || strings.Contains(src, "syntax") {
			t.Errorf("%s does not declare edition 2024:\n%s", path, src)
		}
	}
}
