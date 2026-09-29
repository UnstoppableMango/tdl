package protobuf_test

import (
	"testing"

	"github.com/unstoppablemango/tdl/backend/protobuf"
	"github.com/unstoppablemango/tdl/internal/gen"
	"github.com/unstoppablemango/tdl/plugin"
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

// A primitive of kind type -> type tagged stream marks the request or the
// response it is applied to as streamed.
func TestStreamTaggedArgumentIsStreamed(t *testing.T) {
	const src = `package acme.widgets.v1

primitive Fn: type -> type -> type

primitive Stream: type -> type

type WatchRequest { name: string }

type Widget { name: string }

type Chunk { data: string }

type UploadResponse { size: string }

type WidgetService {
  WatchWidgets: Fn<WatchRequest, Stream<Widget>>
  Upload: Fn<Stream<Chunk>, UploadResponse>
  Chat: Fn<Stream<Chunk>, Stream<Widget>>
}

target protobuf for acme.widgets.v1 {
  Fn => rpc
  Stream => stream
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
		"rpc WatchWidgets(WatchRequest) returns (stream Widget);",
		"rpc Upload(stream Chunk) returns (UploadResponse);",
		"rpc Chat(stream Chunk) returns (stream Widget);",
	)
}

// A service field that is not an rpc skips the service with a warning placed
// at the field, and the messages beside it still generate.
func TestServiceFieldThatIsNotAnRPCIsSkipped(t *testing.T) {
	const src = `package acme.widgets.v1

primitive Fn: type -> type -> type

type GetWidgetRequest { name: string }

type Widget { name: string }

type WidgetService {
  GetWidget: Fn<GetWidgetRequest, Widget>
  label: string
}

target protobuf for acme.widgets.v1 {
  Fn => rpc
  WidgetService => service
}
`
	resp := generateIR(t, lower(t, "widgets.tdl", src, nil))
	diags := resp.GetDiagnostics()
	if len(diags) != 1 {
		t.Fatalf("diagnostics = %+v, want one warning", diags)
	}
	if diags[0].GetSeverity() != plugin.Severity_SEVERITY_WARNING {
		t.Errorf("severity = %v", diags[0].GetSeverity())
	}
	if pos := diags[0].GetPosition(); pos.GetFilename() != "widgets.tdl" || pos.GetLine() != 11 {
		t.Errorf("position = %+v, want widgets.tdl:11, where the field is declared", pos)
	}
	contains(t, diags[0].GetMessage(), "WidgetService", "label", "not an rpc")
	out := compile(t, resp)
	absent(t, out, "service WidgetService", "message WidgetService")
	contains(t, out,
		"message GetWidgetRequest { string name = 1; }",
		"message Widget { string name = 1; }",
	)
}

// A deprecated service carries the option as its first statement, under a
// comment giving the reason.
func TestDeprecatedServiceIsMarked(t *testing.T) {
	const src = `package acme.widgets.v1

primitive Fn: type -> type -> type

type GetWidgetRequest { name: string }

type Widget { name: string }

deprecated("use v2")
type WidgetService { GetWidget: Fn<GetWidgetRequest, Widget> }

target protobuf for acme.widgets.v1 {
  Fn => rpc
  WidgetService => service
}
`
	resp := generateIR(t, lower(t, "widgets.tdl", src, nil))
	if len(resp.GetDiagnostics()) != 0 {
		t.Errorf("diagnostics = %+v", resp.GetDiagnostics())
	}
	contains(t, compile(t, resp),
		"// Deprecated: use v2\nservice WidgetService { option deprecated = true; rpc GetWidget(GetWidgetRequest) returns (Widget); }",
	)
}

// A deprecated rpc field carries the option in the rpc's body.
func TestDeprecatedRPCIsMarked(t *testing.T) {
	const src = `package acme.widgets.v1

primitive Fn: type -> type -> type

type GetWidgetRequest { name: string }

type Widget { name: string }

type WidgetService {
  deprecated GetWidget: Fn<GetWidgetRequest, Widget>
  ListWidgets: Fn<GetWidgetRequest, Widget>
}

target protobuf for acme.widgets.v1 {
  Fn => rpc
  WidgetService => service
}
`
	resp := generateIR(t, lower(t, "widgets.tdl", src, nil))
	if len(resp.GetDiagnostics()) != 0 {
		t.Errorf("diagnostics = %+v", resp.GetDiagnostics())
	}
	out := compile(t, resp)
	contains(t, out,
		"rpc GetWidget(GetWidgetRequest) returns (Widget) { option deprecated = true; }",
		"rpc ListWidgets(GetWidgetRequest) returns (Widget);",
	)
	absent(t, out, "rpc GetWidget(GetWidgetRequest) returns (Widget);")
}

// Doc comments on a service and on its rpc fields are written above them.
func TestServiceDocCommentsAreWritten(t *testing.T) {
	const src = `package acme.widgets.v1

primitive Fn: type -> type -> type

type GetWidgetRequest { name: string }

type Widget { name: string }

/// Serves widgets.
type WidgetService {
  /// Fetches one widget by name.
  GetWidget: Fn<GetWidgetRequest, Widget>
}

target protobuf for acme.widgets.v1 {
  Fn => rpc
  WidgetService => service
}
`
	resp := generateIR(t, lower(t, "widgets.tdl", src, nil))
	if len(resp.GetDiagnostics()) != 0 {
		t.Errorf("diagnostics = %+v", resp.GetDiagnostics())
	}
	contains(t, compile(t, resp),
		"// Serves widgets.\nservice WidgetService {\n  // Fetches one widget by name.\n  rpc GetWidget(GetWidgetRequest) returns (Widget);\n}",
	)
}
