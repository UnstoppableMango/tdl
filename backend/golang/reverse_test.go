package golang_test

import (
	"context"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/unstoppablemango/tdl/ast"
	"github.com/unstoppablemango/tdl/backend/golang"
	"github.com/unstoppablemango/tdl/backend/internal/emit"
	"github.com/unstoppablemango/tdl/internal/sema"
	"github.com/unstoppablemango/tdl/internal/unlower"
	"github.com/unstoppablemango/tdl/ir"
	"github.com/unstoppablemango/tdl/parser"
	"github.com/unstoppablemango/tdl/plugin"
)

// importGo imports files by path, and returns the TDL printed from the
// model, with the response's diagnostics.
func importGo(t *testing.T, files map[string]string) (string, []*plugin.Diagnostic) {
	t.Helper()
	req := &plugin.ImportRequest{Target: golang.Name}
	for path, src := range files {
		req.Files = append(req.Files, &plugin.File{Path: path, Content: []byte(src)})
	}
	resp, err := golang.Backend{}.Import(context.Background(), req)
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
	_, diags := importGo(t, map[string]string{})
	failed(t, diags, "was given none")

	_, diags = importGo(t, map[string]string{"a.go": "package shop\ntype A struct {"})
	failed(t, diags, "does not parse")

	_, diags = importGo(t, map[string]string{"a.go": "package shop", "b.go": "package billing"})
	failed(t, diags, "is package billing")

	_, diags = importGo(t, map[string]string{"a.proto": "syntax = \"proto3\";"})
	failed(t, diags, "reads .go files")

	_, diags = importGo(t, map[string]string{"a.go": "package shop\n\n//tdl:source \"type A {\"\ntype A struct{}\n"})
	failed(t, diags, "is not TDL")
}

// What a handwritten package holds that TDL has no form for, or that
// regenerates differently, warns with its loss code and is read as near
// as TDL comes.
func TestImportWarnsWhatItLoses(t *testing.T) {
	src, diags := importGo(t, map[string]string{
		"shop.go": `package shop

import (
	"io"
	"time"
)

var registry = map[string]Item{}

func New() Item { return Item{} }

type Base struct {
	Created time.Time
}

type Item struct {
	Base
	ID     string
	count  int64
	Small  uint8
	Code   rune
	Size   int
	Grid   [3]int64
	Feed   chan string
	Inline struct{ A string }
	Twice  **string
	Reader io.Reader
	Node   Node
	Nodes  []Node
}

func (i Item) String() string { return i.ID }

type Node interface {
	Children() []Node
}

type Level int

const (
	LevelLow Level = 1
)

type Box[T interface{ Len() int }] struct {
	Item T
}
`,
		"shop_test.go": "package shop\n",
	})
	want := map[string]int{
		// registry, New, Base's embedding, Grid, Feed, Inline, Twice,
		// Item.String, Node, Item.node, Item.nodes, LevelLow, and the test
		// file
		emit.LossUnsupported: 13,
		emit.LossPrimitive:   4, // Small, Code, Size, and Level
		emit.LossName:        1, // count
		emit.LossGeneric:     1, // Box's constraint
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
		"type Item {\n  id: string\n  count: int64\n  small: uint32\n  code: int32\n  size: int64\n  reader: Reader\n}",
		"type Level: int64",
		"type Box<T> {\n  item: T\n}",
		"type Reader { }",
		`Item.id => name("ID")`,
		`Reader => foreign("io", "Reader")`,
	} {
		if !strings.Contains(src, s) {
			t.Errorf("import lacks %q:\n%s", s, src)
		}
	}
}

// A package named main has no TDL package, and any other name is the TDL
// package.
func TestImportReadsThePackageClause(t *testing.T) {
	for clause, want := range map[string]string{"main": "", "shop": "package shop\n"} {
		src, diags := importGo(t, map[string]string{"a.go": "package " + clause + "\n\ntype A struct{ B string }\n"})
		if len(diags) > 0 {
			t.Fatalf("%s: %v", clause, diags)
		}
		if got := strings.HasPrefix(src, "package"); got != (want != "") || !strings.HasPrefix(src, want) {
			t.Errorf("package %s imports as\n%s", clause, src)
		}
	}
}

// One file holding declarations not named for it reads with a file
// directive, and several warn, since they regenerate one per declaration.
func TestImportReadsTheLayout(t *testing.T) {
	src, diags := importGo(t, map[string]string{"models.go": "package shop\n\ntype A struct{ B string }\n"})
	if len(diags) > 0 || !strings.Contains(src, `file("models.go")`) {
		t.Errorf("one file imports as\n%s%v", src, diags)
	}

	src, diags = importGo(t, map[string]string{"a.go": "package shop\n\ntype A struct{ B string }\n"})
	if len(diags) > 0 || strings.Contains(src, "target") {
		t.Errorf("a file per declaration imports as\n%s%v", src, diags)
	}

	_, diags = importGo(t, map[string]string{
		"one.go": "package shop\n\ntype A struct{ B string }\n",
		"two.go": "package shop\n\ntype C struct{ D string }\n",
	})
	if len(diags) != 1 || diags[0].GetCode() != emit.LossUnsupported {
		t.Errorf("diagnostics = %v, want the layout's warning", diags)
	}
}

// A constraint comes back from the message validate writes, and one it
// cannot read warns.
func TestImportReadsConstraintsFromValidate(t *testing.T) {
	src, diags := importGo(t, map[string]string{"a.go": `package shop

import "fmt"

type A struct {
	Pct  int64
	Code string
}

func (a A) validate(path string, errs []error) []error {
	if a.Pct > 100 {
		errs = append(errs, fmt.Errorf("%s.pct: max(100): got %d", path, a.Pct))
	}
	if a.Code != "x: y" {
		errs = append(errs, fmt.Errorf("%s.code: oneOf(\"x: y\", \"100%%\"): not one of them", path))
	}
	if a.Code == "" {
		errs = append(errs, fmt.Errorf("code is empty"))
	}
	return errs
}
`})
	if len(diags) != 1 || diags[0].GetCode() != emit.LossConstraint {
		t.Errorf("diagnostics = %v, want one lossy.constraint", diags)
	}
	for _, s := range []string{"pct: int64 where { max(100) }", `code: string where { oneOf("x: y", "100%") }`} {
		if !strings.Contains(src, s) {
			t.Errorf("import lacks %q:\n%s", s, src)
		}
	}
}

// An annotation's value survives every character a Go string literal
// escapes.
func TestAnnotationsSurviveEscapes(t *testing.T) {
	for _, reason := range []string{`a " b`, `ends in \`, `\\"`, "two\nlines", "tab\there", "café"} {
		escaped := strings.NewReplacer(`\`, `\\`, `"`, `\"`, "\n", `\n`, "\t", `\t`).Replace(reason)
		item := "deprecated(\"" + escaped + "\")\ntype A {\n  b: string\n}"
		src := "package shop\n\n//tdl:source " + strconv.Quote(item) + "\ntype A struct{ B string }\n"
		got, diags := importGo(t, map[string]string{"a.go": src})
		if len(diags) > 0 {
			t.Fatalf("%q: %v", reason, diags)
		}
		if want := `deprecated("` + escaped + `")`; !strings.Contains(got, want) {
			t.Errorf("%q imports as\n%s\nwant %s", reason, got, want)
		}
	}
}

// Annotated output is still a package Go accepts, the package clause of a
// keyword package included.
func TestAnnotatedOutputTypeChecks(t *testing.T) {
	for _, path := range []string{
		"../../testdata/gen/smoke/source.tdl",
		"../../testdata/conformance/keyword-package/source.tdl",
		"../../testdata/roundtrip/go/losses/source.tdl",
	} {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		file, err := parser.Parse(path, strings.NewReader(string(data)))
		if err != nil {
			t.Fatal(err)
		}
		model, diags := sema.Lower(file)
		if len(diags) > 0 {
			t.Fatal(diags)
		}
		model.Targets = append(model.Targets, &ir.TargetBlock{
			Meta:       &ir.Meta{Name: golang.Name},
			ForPackage: model.GetPackage(),
			Directives: []*ir.Directive{{Name: "roundtrip", Target: golang.Name}},
		})
		resp, err := golang.Backend{}.Generate(context.Background(), &plugin.Request{Target: golang.Name, Model: model})
		if err != nil {
			t.Fatal(err)
		}
		if len(resp.GetDiagnostics()) > 0 {
			t.Errorf("%s: diagnostics = %v", path, resp.GetDiagnostics())
		}
		files(t, resp)
	}
}
