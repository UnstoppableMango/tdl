package lsp_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

// TestConformanceCorpusPublishesNothing checks every conformance case
// publishes no diagnostic.
func TestConformanceCorpusPublishesNothing(t *testing.T) {
	for _, dir := range subdirs(t, "../../testdata/conformance") {
		t.Run(filepath.Base(dir), func(t *testing.T) {
			path, src := corpusFile(t, dir)

			diags := newSession(t).open(path, src)
			for _, d := range diags {
				t.Errorf("unexpected diagnostic at %d:%d: %s",
					d.Range.Start.Line+1, d.Range.Start.Character+1, message(t, d))
			}
		})
	}
}

// TestInvalidCorpusPublishesTheError checks every invalid case publishes a
// diagnostic containing its error.golden.
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
				if strings.Contains(message(t, d), want) {
					return
				}
			}
			t.Errorf("no diagnostic contains %q; got %v", want, messages(t, diags))
		})
	}
}

// TestFixingAnErrorClearsIt checks the server publishes an empty list for a
// document that no longer has errors.
func TestFixingAnErrorClearsIt(t *testing.T) {
	const broken = "package p\n\ntype User: Entity {\n  id: Nope\n}\n"
	const fixed = "package p\n\ntype User: Entity {\n  id: string\n}\n"

	s := newSession(t)
	path := abs(t, "fix.tdl")

	if diags := s.open(path, broken); len(diags) == 0 {
		t.Fatal("expected a diagnostic for the undefined type")
	}
	if diags := s.change(path, fixed, 2); len(diags) != 0 {
		t.Errorf("expected the diagnostic to clear, got %v", messages(t, diags))
	}
}

// TestSyntaxErrorsSuppressLoweringDiagnostics checks a file that does not
// parse publishes syntax errors only.
func TestSyntaxErrorsSuppressLoweringDiagnostics(t *testing.T) {
	// The `{` is never closed, and `Missing` is undefined.
	const src = "package p\n\ntype User: Entity {\n  id: Missing\n"

	diags := newSession(t).open(abs(t, "broken.tdl"), src)
	if len(diags) == 0 {
		t.Fatal("expected a syntax error")
	}
	for _, d := range diags {
		if msg := message(t, d); strings.Contains(msg, "undefined") {
			t.Errorf("lowering diagnostic published for a file that does not parse: %s", msg)
		}
	}
}

// TestDiagnosticsUseUTF16Columns checks columns are converted from bytes
// to UTF-16 code units. The string default holds a two-byte rune and one
// outside the basic multilingual plane, on the same line as the error.
func TestDiagnosticsUseUTF16Columns(t *testing.T) {
	const src = "package p\n\ntype User: Entity {\n  a: string = \"né 🙂\" b: Nope\n}\n"

	diags := newSession(t).open(abs(t, "utf16.tdl"), src)
	if len(diags) != 1 {
		t.Fatalf("expected one diagnostic, got %v", messages(t, diags))
	}

	// `Nope` is 28 bytes and 25 UTF-16 code units into line 3.
	got := diags[0].Range
	if got.Start.Line != 3 || got.Start.Character != 25 {
		t.Errorf("start = %d:%d, want 3:25", got.Start.Line, got.Start.Character)
	}
	if got.End.Line != 3 || got.End.Character != 29 {
		t.Errorf("end = %d:%d, want 3:29", got.End.Line, got.End.Character)
	}
}

// TestDiagnosticsRangeOverTheNode checks a diagnostic about a node on one
// line covers all of it, not only its first word.
func TestDiagnosticsRangeOverTheNode(t *testing.T) {
	const src = "package p\n\ntype User: Entity {\n  a: int where { min(\"two words\") }\n}\n"

	diags := newSession(t).open(abs(t, "span.tdl"), src)
	if len(diags) != 1 {
		t.Fatalf("expected one diagnostic, got %v", messages(t, diags))
	}

	// `"two words"` starts 21 bytes into line 3 and is 11 long.
	want := protocol.Range{
		Start: protocol.Position{Line: 3, Character: 21},
		End:   protocol.Position{Line: 3, Character: 32},
	}
	if got := diags[0].Range; got != want {
		t.Errorf("range = %v, want %v (%s)", got, want, message(t, diags[0]))
	}
}

// TestDefinitionJumpsToADeclaration checks the range covers the declared
// name rather than the `type` keyword.
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
		t.Fatalf("unexpected diagnostics: %v", messages(t, diags))
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

// TestDefinitionCrossesAnImport jumps into a `_` import's file that is on
// disk and not open.
func TestDefinitionCrossesAnImport(t *testing.T) {
	dir := t.TempDir()
	dep := filepath.Join(dir, "common.tdl")
	writeFile(t, dep, "package common\n\nprimitive string\n\ntype Money {\n  amount: string\n}\n")

	src := "package p\n\n" +
		"import \"common.tdl\" as _\n\n" +
		"type Order: Entity {\n" +
		"  id: string\n" +
		"  total: Money\n" +
		"}\n"
	path := filepath.Join(dir, "order.tdl")
	writeFile(t, path, src)

	s := newSession(t)
	if diags := s.open(path, src); len(diags) != 0 {
		t.Fatalf("unexpected diagnostics: %v", messages(t, diags))
	}

	locs := s.definition(path, src, "Money\n}")
	if len(locs) != 1 {
		t.Fatalf("expected one location, got %d", len(locs))
	}
	if got := locs[0].URI; got != uri.File(dep) {
		t.Errorf("uri = %s, want %s", got, uri.File(dep))
	}

	// `Money` is declared on line 5 of the dependency, five characters in.
	want := protocol.Range{
		Start: protocol.Position{Line: 4, Character: 5},
		End:   protocol.Position{Line: 4, Character: 10},
	}
	if locs[0].Range != want {
		t.Errorf("range = %v, want %v", locs[0].Range, want)
	}
}

func TestDefinitionReturnsNothing(t *testing.T) {
	const src = "package p\n\n" +
		"primitive string\n\n" +
		"type User: Entity {\n" +
		"  id: string\n" +
		"  email: Nope\n" +
		"}\n"

	cases := map[string]string{
		// The prelude is embedded and has no file to open.
		"the prelude":     "Entity",
		"an unknown name": "Nope",
		// `type` is a keyword.
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

func writeFile(t *testing.T, path, src string) {
	t.Helper()

	if err := os.WriteFile(path, []byte(src), 0o600); err != nil {
		t.Fatalf("writing %s: %v", path, err)
	}
}

func messages(t *testing.T, diags []protocol.Diagnostic) []string {
	t.Helper()

	out := make([]string, 0, len(diags))
	for _, d := range diags {
		out = append(out, message(t, d))
	}
	return out
}

// abs resolves a name against the working directory, since a document URI
// is absolute.
func abs(t *testing.T, name string) string {
	t.Helper()

	path, err := filepath.Abs(name)
	if err != nil {
		t.Fatalf("resolving %s: %v", name, err)
	}
	return path
}
