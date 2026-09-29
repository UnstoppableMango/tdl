package protobuf_test

import (
	"testing"

	"github.com/unstoppablemango/tdl/backend/protobuf"
	"github.com/unstoppablemango/tdl/internal/gen"
)

// A service is declared by tags: service on the type holding the methods,
// and rpc on the primitive each method's field is typed with.
func TestTaggedServiceIsEmitted(t *testing.T) {
	const src = `package acme.widgets.v1

primitive Fn: type -> type -> type

type GetWidgetRequest { name: string }

type Widget { name: string }

type WidgetService { GetWidget: Fn<GetWidgetRequest, Widget> }

target protobuf for acme.widgets.v1 {
  Fn => rpc
  WidgetService => service
}
`
	model := lower(t, "widgets.tdl", src, nil)
	if problems := gen.CheckDirectives(protobuf.Name, model, protobuf.Backend{}.Describe()); len(problems) != 0 {
		t.Errorf("directive problems = %+v", problems)
	}

	resp := generateIR(t, model)
	if len(resp.GetDiagnostics()) != 0 {
		t.Errorf("diagnostics = %+v", resp.GetDiagnostics())
	}
	out := compile(t, resp)
	contains(t, out,
		"service WidgetService { rpc GetWidget(GetWidgetRequest) returns (Widget); }",
		"message GetWidgetRequest { string name = 1; }",
		"message Widget { string name = 1; }",
	)
	absent(t, out, "message WidgetService")
}
