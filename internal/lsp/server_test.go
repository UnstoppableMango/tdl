package lsp_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

// TestConformanceCorpusPublishesNothing asks the server the question
// internal/sema/corpus_test.go asks lowering: every case in the corpus
// lowers with no diagnostic at all.
//
// Asking it through the protocol is what makes it a server test. The
// corpus is plain text rather than Go code, so adding a case adds a case
// here the way it adds one to the parser's tests.
func TestConformanceCorpusPublishesNothing(t *testing.T) {
	for _, dir := range subdirs(t, "../../testdata/conformance") {
		t.Run(filepath.Base(dir), func(t *testing.T) {
			path, src := corpusFile(t, dir)

			diags := newSession(t).open(path, src)
			for _, d := range diags {
				t.Errorf("unexpected diagnostic at %d:%d: %s",
					d.Range.Start.Line+1, d.Range.Start.Character+1, d.Message)
			}
		})
	}
}

// TestInvalidCorpusPublishesTheError is the other half: every case in
// testdata/invalid produces at least one diagnostic, carrying the text in
// the sibling error.golden.
func TestInvalidCorpusPublishesTheError(t *testing.T) {
	for _, dir := range subdirs(t, "../../testdata/invalid") {
		t.Run(filepath.Base(dir), func(t *testing.T) {
			path, src := corpusFile(t, dir)

			golden, err := os.ReadFile(filepath.Join(dir, "error.golden"))
			if err != nil {
				t.Fatalf("reading error.golden: %v", err)
			}
			want := strings.TrimSpace(string(golden))

			diags := newSession(t).open(path, src)
			if len(diags) == 0 {
				t.Fatal("expected a diagnostic, got none")
			}

			for _, d := range diags {
				if strings.Contains(d.Message, want) {
					return
				}
			}
			t.Errorf("no diagnostic contains %q; got %v", want, messages(diags))
		})
	}
}

// TestFixingAnErrorClearsIt is the property an editor actually shows: a
// file that stops being wrong stops being underlined.
//
// The server publishes for the document it analyzed even when it found
// nothing, which is what retracts the last report; a server that published
// only what it found would leave the first error on screen forever.
func TestFixingAnErrorClearsIt(t *testing.T) {
	const broken = "package p\n\ntype User: Entity {\n  id: Nope\n}\n"
	const fixed = "package p\n\ntype User: Entity {\n  id: string\n}\n"

	s := newSession(t)
	path := abs(t, "fix.tdl")

	if diags := s.open(path, broken); len(diags) == 0 {
		t.Fatal("expected a diagnostic for the undefined type")
	}
	if diags := s.change(path, fixed, 2); len(diags) != 0 {
		t.Errorf("expected the diagnostic to clear, got %v", messages(diags))
	}
}

// TestSyntaxErrorsSuppressLoweringDiagnostics holds the rule
// docs/design/lsp.md states: a file that does not parse publishes syntax
// errors only.
//
// Lowering a tree with holes in it reports names that are undefined
// because the declaration naming them failed to parse, and reporting those
// beside the syntax error buries it.
func TestSyntaxErrorsSuppressLoweringDiagnostics(t *testing.T) {
	// The `{` is never closed, so the declaration does not parse and the
	// type it names is never declared.
	const src = "package p\n\ntype User: Entity {\n  id: Missing\n"

	diags := newSession(t).open(abs(t, "broken.tdl"), src)
	if len(diags) == 0 {
		t.Fatal("expected a syntax error")
	}
	for _, d := range diags {
		if strings.Contains(d.Message, "undefined") {
			t.Errorf("lowering diagnostic published for a file that does not parse: %s", d.Message)
		}
	}
}

// TestDiagnosticsUseUTF16Columns is the assertion that fails silently if
// nothing makes it.
//
// lex.Position counts a column in bytes and the protocol counts a
// character in UTF-16 code units, so a diagnostic after a non-ASCII
// character lands in the wrong place unless something converts.
//
// The two have to be on the same line for the difference to show, which is
// what the second field on this one is for: whitespace is insignificant
// and an item ends where the next begins, so a field carrying a string
// default and a field naming an undefined type share a line. The string
// holds a two-byte rune and a rune outside the basic multilingual plane,
// which are the two cases that diverge: 2 bytes for 1 unit, and 4 bytes
// for 2.
func TestDiagnosticsUseUTF16Columns(t *testing.T) {
	const src = "package p\n\ntype User: Entity {\n  a: string = \"né 🙂\" b: Nope\n}\n"

	diags := newSession(t).open(abs(t, "utf16.tdl"), src)
	if len(diags) != 1 {
		t.Fatalf("expected one diagnostic, got %v", messages(diags))
	}

	// `Nope` is 28 bytes into line 3 and 25 UTF-16 code units into it, so
	// a server that published the byte column would be off by three.
	got := diags[0].Range
	if got.Start.Line != 3 || got.Start.Character != 25 {
		t.Errorf("start = %d:%d, want 3:25", got.Start.Line, got.Start.Character)
	}
	if got.End.Line != 3 || got.End.Character != 29 {
		t.Errorf("end = %d:%d, want 3:29", got.End.Line, got.End.Character)
	}
}

// TestDefinitionJumpsToADeclaration is the feature: a cursor on a type in
// a field lands on the name that declares it.
//
// The range covers the name rather than the keyword that declares it. A
// declaration's position is where it starts, which is `type`, and an
// editor highlights whatever range it is handed.
func TestDefinitionJumpsToADeclaration(t *testing.T) {
	const src = "package p\n\n" +
		"primitive string\n\n" +
		"type Email: string\n\n" +
		"type User: Entity {\n" +
		"  id: string\n" +
		"  email: Email\n" +
		"}\n"

	s := newSession(t)
	path := abs(t, "def.tdl")
	if diags := s.open(path, src); len(diags) != 0 {
		t.Fatalf("unexpected diagnostics: %v", messages(diags))
	}

	locs := s.definition(path, src, "Email\n}")
	if len(locs) != 1 {
		t.Fatalf("expected one location, got %d", len(locs))
	}
	if got := locs[0].URI; got != uri.File(path) {
		t.Errorf("uri = %s, want %s", got, uri.File(path))
	}

	// `Email` is declared on line 5, five characters in.
	want := protocol.Range{
		Start: protocol.Position{Line: 4, Character: 5},
		End:   protocol.Position{Line: 4, Character: 10},
	}
	if locs[0].Range != want {
		t.Errorf("range = %v, want %v", locs[0].Range, want)
	}
}

// TestDefinitionCrossesAnImport is the same jump into another file, which
// is the case the index answers and nothing in internal/lsp decides: a `_`
// import merges the dependency's names in, and sema records where the
// dependency declares each one.
//
// The dependency is on disk and not open, which is the normal case: a
// person jumps into a file to read it and has not opened it first.
func TestDefinitionCrossesAnImport(t *testing.T) {
	dir := t.TempDir()
	dep := filepath.Join(dir, "common.tdl")
	writeFile(t, dep, "package common\n\nprimitive string\n\ntype Money {\n  amount: string\n}\n")

	src := "package p\n\n" +
		"import \"common.tdl\" as _\n\n" +
		"primitive string\n\n" +
		"type Order: Entity {\n" +
		"  id: string\n" +
		"  total: Money\n" +
		"}\n"
	path := filepath.Join(dir, "order.tdl")
	writeFile(t, path, src)

	s := newSession(t)
	if diags := s.open(path, src); len(diags) != 0 {
		t.Fatalf("unexpected diagnostics: %v", messages(diags))
	}

	locs := s.definition(path, src, "Money\n}")
	if len(locs) != 1 {
		t.Fatalf("expected one location, got %d", len(locs))
	}
	if got := locs[0].URI; got != uri.File(dep) {
		t.Errorf("uri = %s, want %s", got, uri.File(dep))
	}

	// `Money` is declared on line 5 of the dependency, five characters in,
	// which the server knows only because it read the file.
	want := protocol.Range{
		Start: protocol.Position{Line: 4, Character: 5},
		End:   protocol.Position{Line: 4, Character: 10},
	}
	if locs[0].Range != want {
		t.Errorf("range = %v, want %v", locs[0].Range, want)
	}
}

// TestDefinitionReturnsNothing covers the three answers that are not a
// location, all of which are an answer rather than a failure.
func TestDefinitionReturnsNothing(t *testing.T) {
	const src = "package p\n\n" +
		"primitive string\n\n" +
		"type User: Entity {\n" +
		"  id: string\n" +
		"  email: Nope\n" +
		"}\n"

	cases := map[string]string{
		// The prelude is embedded under a name that is no path, so there
		// is no file to open. docs/design/lsp.md argues hover instead.
		// `Entity` is the prelude's class this entity conforms to.
		"the prelude":     "Entity",
		"an unknown name": "Nope",
		// `type` is a keyword, and a keyword is not a name the index has.
		"not a name": "type User",
	}

	s := newSession(t)
	path := abs(t, "nothing.tdl")
	if diags := s.open(path, src); len(diags) == 0 {
		t.Fatal("expected a diagnostic for the undefined type")
	}

	for name, needle := range cases {
		t.Run(name, func(t *testing.T) {
			if locs := s.definition(path, src, needle); len(locs) != 0 {
				t.Errorf("expected no location, got %v", locs)
			}
		})
	}
}

// writeFile puts a file where an import can find it.
func writeFile(t *testing.T, path, src string) {
	t.Helper()

	if err := os.WriteFile(path, []byte(src), 0o600); err != nil {
		t.Fatalf("writing %s: %v", path, err)
	}
}

// messages is what a failure prints.
func messages(diags []protocol.Diagnostic) []string {
	out := make([]string, 0, len(diags))
	for _, d := range diags {
		out = append(out, d.Message)
	}
	return out
}

// abs resolves a name against the working directory, because a document
// URI is absolute and an import resolves relative to the file that wrote
// it.
func abs(t *testing.T, name string) string {
	t.Helper()

	path, err := filepath.Abs(name)
	if err != nil {
		t.Fatalf("resolving %s: %v", name, err)
	}
	return path
}
