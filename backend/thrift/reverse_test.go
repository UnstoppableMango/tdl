package thrift_test

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/unstoppablemango/tdl/ast"
	"github.com/unstoppablemango/tdl/backend/internal/emit"
	"github.com/unstoppablemango/tdl/backend/thrift"
	"github.com/unstoppablemango/tdl/internal/unlower"
	"github.com/unstoppablemango/tdl/plugin"
)

// importThrift imports one file and returns the TDL printed from the
// model, with the response's diagnostics.
func importThrift(t *testing.T, files ...string) (string, []*plugin.Diagnostic) {
	t.Helper()
	req := &plugin.ImportRequest{Target: thrift.Name}
	for _, src := range files {
		req.Files = append(req.Files, &plugin.File{Path: "shop.thrift", Content: []byte(src)})
	}
	resp, err := thrift.Backend{}.Import(context.Background(), req)
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	if resp.GetModel() == nil {
		return "", resp.GetDiagnostics()
	}
	return ast.Fprint(unlower.File(resp.GetModel())), resp.GetDiagnostics()
}

func failed(t *testing.T, diags []*plugin.Diagnostic, want string) {
	t.Helper()
	if len(diags) != 1 || diags[0].GetSeverity() != plugin.Severity_SEVERITY_ERROR || !strings.Contains(diags[0].GetMessage(), want) {
		t.Errorf("diagnostics = %v, want one error containing %q", diags, want)
	}
}

func TestImportRefusesWhatItCannotRead(t *testing.T) {
	_, diags := importThrift(t, "struct A {}", "struct B {}")
	failed(t, diags, "one .thrift file")

	_, diags = importThrift(t, "struct A {")
	failed(t, diags, "does not parse")

	_, diags = importThrift(t, "include \"other.thrift\"\nstruct A {}")
	failed(t, diags, "includes are not imported yet")
}

// What a handwritten file holds that TDL has no form for, or that
// regenerates differently, warns with its loss code and is read as near as
// TDL comes.
func TestImportWarnsWhatItLoses(t *testing.T) {
	src, diags := importThrift(t, `namespace * shop
namespace go shop.gen

const i32 MAX = 10

enum Kind {
  zero = 0
  one = 1
}

exception NotFound {
  1: string message
}

union Value {
  1: string text
  2: i64 number
}

struct Item {
  1: required string id
  2: i16 count = 1
  3: byte flag
  4: Value value
  5: Kind kind
  6: string note (go.tag = "json")
  -1: string bare
}

struct ByCard {
  1: string last4
}

union Payment {
  1: ByCard by_card
}

service Shop {
  Item get(1: string id) throws (1: NotFound nf)
}
`)
	want := map[string]int{
		emit.LossUnsupported: 8, // namespace, const, exception, union, value, required, go.tag, service
		emit.LossNumber:      2, // Kind's zero, Item's bare field
		emit.LossDefault:     1,
		emit.LossPrimitive:   2,
		emit.LossName:        1, // Payment's by_card member
	}
	got := map[string]int{}
	for _, d := range diags {
		got[d.GetCode()]++
	}
	for code, n := range want {
		if got[code] != n {
			t.Errorf("%d %s warnings, want %d: %v", got[code], code, n, diags)
		}
	}
	for _, s := range []string{
		"enum Kind { Zero One }",
		"type NotFound {",
		"id: string",
		"count: int32",
		"flag: int32",
		"kind: Kind",
		"note: string",
		"enum Payment {\n  ByCard { last4: string }\n}",
		`Kind.Zero => name("zero")`,
	} {
		if !strings.Contains(src, s) {
			t.Errorf("import lacks %q:\n%s", s, src)
		}
	}
	if strings.Contains(src, "value:") {
		t.Errorf("a field naming a union that is not read is kept:\n%s", src)
	}
}

// A union whose member struct something else names is not a fielded enum.
func TestImportKeepsAStructUsedElsewhere(t *testing.T) {
	src, diags := importThrift(t, `namespace * shop
struct Card { 1: string last4 }
union Payment { 1: Card card }
struct Wallet { 1: Card card }
`)
	if !strings.Contains(src, "type Card {") || strings.Contains(src, "enum Payment") {
		t.Errorf("import:\n%s", src)
	}
	if !slices.ContainsFunc(diags, func(d *plugin.Diagnostic) bool { return strings.Contains(d.GetMessage(), "union Payment") }) {
		t.Errorf("diagnostics = %v", diags)
	}
}

// An annotation value survives backslashes and quotes, which thriftgo
// unescapes only in part.
func TestAnnotationsSurviveEscapes(t *testing.T) {
	for _, reason := range []string{`a \" b`, `ends in \`, `\\"`, `50%5C off`, "two\nlines"} {
		src := "namespace * shop\nstruct A {} (deprecated = \"\", tdl.doc = \"\", tdl.reason = " + thrift.Literal(reason) + ")\n"
		got, diags := importThrift(t, src)
		if len(diags) > 0 {
			t.Fatalf("%q: %v", reason, diags)
		}
		escaped := strings.NewReplacer(`\`, `\\`, `"`, `\"`, "\n", `\n`).Replace(reason)
		if want := `deprecated("` + escaped + `")`; !strings.Contains(got, want) {
			t.Errorf("%q imports as\n%s\nwant %s", reason, got, want)
		}
	}
}
