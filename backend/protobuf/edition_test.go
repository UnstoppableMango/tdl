package protobuf_test

import (
	"testing"

	"github.com/unstoppablemango/tdl/backend/internal/irtest"
	"github.com/unstoppablemango/tdl/backend/protobuf"
	"github.com/unstoppablemango/tdl/internal/gen"
	"github.com/unstoppablemango/tdl/ir"
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
