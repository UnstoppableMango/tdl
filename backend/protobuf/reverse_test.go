package protobuf_test

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/unstoppablemango/tdl/ast"
	"github.com/unstoppablemango/tdl/backend/protobuf"
	"github.com/unstoppablemango/tdl/internal/unlower"
	"github.com/unstoppablemango/tdl/plugin"
)

// importProto imports files by path and returns the TDL printed from the
// model, with the response's diagnostics.
func importProto(t *testing.T, files map[string]string) (string, []*plugin.Diagnostic) {
	t.Helper()
	req := &plugin.ImportRequest{Target: protobuf.Name}
	for p, src := range files {
		req.Files = append(req.Files, &plugin.File{Path: p, Content: []byte(src)})
	}
	slices.SortFunc(req.Files, func(a, b *plugin.File) int { return strings.Compare(a.GetPath(), b.GetPath()) })
	resp, err := protobuf.Backend{}.Import(context.Background(), req)
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	if resp.GetModel() == nil {
		return "", resp.GetDiagnostics()
	}
	return ast.Fprint(unlower.File(resp.GetModel())), resp.GetDiagnostics()
}

func codes(diags []*plugin.Diagnostic) []string {
	var out []string
	for _, d := range diags {
		out = append(out, d.GetCode())
	}
	return out
}

func TestImportRefusesTwoPackages(t *testing.T) {
	_, diags := importProto(t, map[string]string{
		"a/a.proto": "syntax = \"proto3\";\npackage a;\nmessage A {}\n",
		"b/b.proto": "syntax = \"proto3\";\npackage b;\nmessage B {}\n",
	})
	if len(diags) != 1 || diags[0].GetSeverity() != plugin.Severity_SEVERITY_ERROR || !strings.Contains(diags[0].GetMessage(), "one package at a time") {
		t.Errorf("diagnostics = %v", diags)
	}
}

func TestImportReportsWhatDoesNotCompile(t *testing.T) {
	_, diags := importProto(t, map[string]string{"a/a.proto": "syntax = \"proto3\";\npackage a;\nmessage A { Missing m = 1; }\n"})
	if len(diags) != 1 || diags[0].GetSeverity() != plugin.Severity_SEVERITY_ERROR || !strings.Contains(diags[0].GetMessage(), "do not compile") {
		t.Errorf("diagnostics = %v", diags)
	}
}

// A nested type is hoisted, an integer encoding is dropped, and a wrapper
// is optional, each with a warning.
func TestImportWarnsWhatDoesNotRegenerate(t *testing.T) {
	src, diags := importProto(t, map[string]string{"a/a.proto": `syntax = "proto3";
package a;
import "google/protobuf/wrappers.proto";
message Outer {
  message Inner { string x = 1; }
  enum Kind { KIND_UNSPECIFIED = 0; KIND_BIG = 1; }
  Inner inner = 1;
  Kind kind = 2;
  sint64 delta = 3;
  google.protobuf.StringValue label = 4;
}
service Svc {}
`})
	want := []string{"lossy.primitive", "lossy.optional", "lossy.unsupported", "lossy.unsupported", "lossy.unsupported"}
	if got := codes(diags); !slices.Equal(slices.Sorted(slices.Values(got)), slices.Sorted(slices.Values(want))) {
		t.Errorf("codes = %v, want %v\n%v", got, want, diags)
	}
	contains(t, src, "type Outer { inner: Inner kind: Kind delta: int64 label: string? }", "type Inner { x: string }", "enum Kind { Big }")
}

// Hand-written names that the convention does not produce keep a name
// directive, and numbers allocation would not give keep a pin.
func TestImportKeepsNamesAndNumbers(t *testing.T) {
	src, _ := importProto(t, map[string]string{"a/a.proto": `syntax = "proto3";
package a;
message user_record {
  string ID = 3;
  string name = 1;
}
`})
	contains(t, src,
		"type user_record {",
		`user_record => name("user_record")`,
		`user_record.id => name("ID")`,
		"user_record.id => number(3)")
	absent(t, src, "user_record.name => number")
}
