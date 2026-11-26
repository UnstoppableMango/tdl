package salesforce_test

import (
	"context"
	"strings"
	"testing"

	"github.com/unstoppablemango/tdl/ast"
	"github.com/unstoppablemango/tdl/backend/salesforce"
	"github.com/unstoppablemango/tdl/internal/analysis"
	"github.com/unstoppablemango/tdl/internal/sema"
	"github.com/unstoppablemango/tdl/internal/unlower"
	"github.com/unstoppablemango/tdl/parser"
	"github.com/unstoppablemango/tdl/plugin"
)

// generateTDL generates src, failing on any diagnostic.
func generateTDL(t *testing.T, src string) []*plugin.File {
	t.Helper()
	file, err := parser.Parse("source.tdl", strings.NewReader(src))
	if err != nil {
		t.Fatal(err)
	}
	model, diags := sema.Lower(file)
	if len(diags) > 0 {
		t.Fatal(diags)
	}
	b := salesforce.Backend{}
	req, problems := analysis.Request(model, b.Describe(), salesforce.Name, "out")
	if len(problems) > 0 {
		t.Fatal(problems)
	}
	resp, err := b.Generate(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range resp.GetDiagnostics() {
		t.Errorf("generating: %s", d.GetMessage())
	}
	return resp.GetFiles()
}

// readBack imports files and prints the model as TDL.
func readBack(t *testing.T, files []*plugin.File) (string, []*plugin.Diagnostic) {
	t.Helper()
	resp, err := salesforce.ReadObjects(files, true)
	if err != nil {
		t.Fatal(err)
	}
	if resp.GetModel() == nil {
		return "", resp.GetDiagnostics()
	}
	return ast.Fprint(unlower.File(resp.GetModel())), resp.GetDiagnostics()
}

// comesBack generates src and holds what it reads back to src.
func comesBack(t *testing.T, src string) {
	t.Helper()
	got, diags := readBack(t, generateTDL(t, src))
	for _, d := range diags {
		if !strings.Contains(d.GetMessage(), "Apex classes are not read yet") {
			t.Errorf("import warns: %s", d.GetMessage())
		}
	}
	if got != src {
		t.Errorf("reads back as\n%s\nwant\n%s", got, src)
	}
}

// What generating writes for an object reads back as the source.
func TestObjectsReadBack(t *testing.T) {
	comesBack(t, `package salesforce

/// A customer.
///
/// It has two paragraphs.
type Customer: Entity { }

deprecated("use Customer")
type Client: Entity { }

type LineItem: Entity { }

target salesforce for salesforce {
  LineItem => name("Line_Item")
  LineItem => label("Line")
  LineItem => plural("Line items")
}
`)
}
