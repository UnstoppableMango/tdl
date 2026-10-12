package smithy_test

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/unstoppablemango/tdl/ast"
	"github.com/unstoppablemango/tdl/backend/internal/emit"
	"github.com/unstoppablemango/tdl/backend/smithy"
	"github.com/unstoppablemango/tdl/internal/unlower"
	"github.com/unstoppablemango/tdl/plugin"
)

// importSmithy imports files by path, and returns the TDL printed from the
// model, with the response's diagnostics. It skips without the Smithy
// CLI, which import runs.
func importSmithy(t *testing.T, files map[string]string) (string, []*plugin.Diagnostic) {
	t.Helper()
	if _, err := smithy.CLI(); err != nil {
		t.Skip(err)
	}
	req := &plugin.ImportRequest{Target: smithy.Name}
	for path, src := range files {
		req.Files = append(req.Files, &plugin.File{Path: path, Content: []byte(src)})
	}
	slices.SortFunc(req.Files, func(a, b *plugin.File) int { return strings.Compare(a.GetPath(), b.GetPath()) })
	resp, err := smithy.Backend{}.Import(context.Background(), req)
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
	_, diags := importSmithy(t, map[string]string{})
	failed(t, diags, "was given none")

	_, diags = importSmithy(t, map[string]string{"a.ts": ""})
	failed(t, diags, "reads .smithy files")

	_, diags = importSmithy(t, map[string]string{"a.smithy": "$version: \"2\"\nnamespace a\nstructure A {"})
	failed(t, diags, "rejects the files")

	_, diags = importSmithy(t, map[string]string{
		"a.smithy": "$version: \"2\"\nnamespace a\nstructure A {}\n",
		"b.smithy": "$version: \"2\"\nnamespace b\nstructure B {}\n",
	})
	failed(t, diags, "one namespace")

	_, diags = importSmithy(t, map[string]string{"a.smithy": "$version: \"2\"\nmetadata tdl = {}\nnamespace a\n@tdl#source(\"type A {\")\nstructure A {}\n"})
	failed(t, diags, "is not TDL")
}

// What a handwritten model holds that TDL has no form for, or that
// regenerates differently, warns with its loss code and is read as near as
// TDL comes.
func TestImportWarnsWhatItLoses(t *testing.T) {
	src, diags := importSmithy(t, map[string]string{"shop.smithy": `$version: "2"

metadata owner = "shop team"

namespace shop

structure Item {
    @required
    @length(min: 1)
    id: String
    count: Short
    blob: Document
}

intEnum Level {
    LOW = 1
    HIGH = 2
}

enum Color {
    RED = "red-ish"
}

structure Shared {
    @required
    id: String
}

union Either {
    shared: Shared
}

structure Holder {
    @required
    shared: Shared
}

union Odd {
    SSN: OddSSN
}

structure OddSSN {
    @required
    value: String
}

service Shop {
    version: "1"
}
`})

	for _, want := range []string{
		"type Item {\n  id: string\n  count: int32?\n}",
		"enum Level { Low High }",
		"enum Color { Red }",
		"type Holder {\n  shared: Shared\n}",
		"enum Odd {\n  SSN { value: string }\n}",
	} {
		if !strings.Contains(src, want) {
			t.Errorf("import does not hold\n%s\nin\n%s", want, src)
		}
	}
	if strings.Contains(src, "Either") || strings.Contains(src, "Shop") {
		t.Errorf("import reads what TDL has no form for:\n%s", src)
	}

	var codes []string
	for _, d := range diags {
		codes = append(codes, d.GetCode())
	}
	slices.Sort(codes)
	codes = slices.Compact(codes)
	if want := []string{emit.LossName, emit.LossPrimitive, emit.LossUnsupported}; !slices.Equal(codes, want) {
		t.Errorf("codes = %v, want %v: %v", codes, want, diags)
	}
	for _, want := range []string{
		"metadata owner",
		"@smithy.api#length",
		"is a Short",
		"is a Document",
		"intEnum Level",
		`value "red-ish"`,
		"union Either",
		"member SSN",
		"service Shop",
	} {
		if !slices.ContainsFunc(diags, func(d *plugin.Diagnostic) bool { return strings.Contains(d.GetMessage(), want) }) {
			t.Errorf("no warning says %q: %v", want, diags)
		}
	}
}

// A shape written by hand under the names the generator derives reads back
// as the declarations it would write, so the files regenerate.
func TestImportReadsGeneratedNames(t *testing.T) {
	src, diags := importSmithy(t, map[string]string{"shop.smithy": `$version: "2"

namespace shop

structure orderLine {
    @required
    tags: StringSet
    byKey: SparseStringLongMap
}

@uniqueItems
list StringSet {
    member: String
}

@sparse
map SparseStringLongMap {
    key: String
    value: Long
}

/// Not the shape a member would use.
list StringList {
    member: String
}
`})
	if len(diags) > 0 {
		t.Errorf("diagnostics = %v", diags)
	}
	for _, want := range []string{
		"type orderLine {\n  tags: {string}\n  byKey: {string -> int64?}?\n}",
		"/// Not the shape a member would use.\ntype StringList: [string]",
		`orderLine => name("orderLine")`,
	} {
		if !strings.Contains(src, want) {
			t.Errorf("import does not hold\n%s\nin\n%s", want, src)
		}
	}
}
