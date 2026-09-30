package protobuf_test

import (
	"testing"

	"github.com/unstoppablemango/tdl/backend/protobuf"
	"github.com/unstoppablemango/tdl/internal/gen"
)

// The rpc and stream primitives may be declared once in a shared file and
// merged in with a `_` import; the tags attach to their externs, and the
// service is emitted as if they were declared locally.
func TestServiceWithImportedRPCPrimitives(t *testing.T) {
	const rpcSource = `package shim.rpc

primitive Fn: type -> type -> type

primitive Stream: type -> type
`
	const src = `package acme.widgets.v1

import "rpc.tdl" as _

type GetWidgetRequest { name: string }

type Widget { name: string }

type WidgetService {
  GetWidget: Fn<GetWidgetRequest, Widget>
  WatchWidgets: Fn<GetWidgetRequest, Stream<Widget>>
}

target protobuf for acme.widgets.v1 {
  Fn => rpc
  Stream => stream
  WidgetService => service
}
`
	model := lower(t, "main.tdl", src, sources{"rpc.tdl": rpcSource})
	if problems := gen.CheckDirectives(protobuf.Name, model, protobuf.Backend{}.Describe()); len(problems) != 0 {
		t.Errorf("directive problems = %+v", problems)
	}

	resp := generateIR(t, model)
	if len(resp.GetDiagnostics()) != 0 {
		t.Errorf("diagnostics = %+v", resp.GetDiagnostics())
	}
	out := compile(t, resp)
	contains(t, out,
		"rpc GetWidget(GetWidgetRequest) returns (Widget);",
		"rpc WatchWidgets(GetWidgetRequest) returns (stream Widget);",
		"message GetWidgetRequest { string name = 1; }",
		"message Widget { string name = 1; }",
	)
	absent(t, out, "message WidgetService")
}
