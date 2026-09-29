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
