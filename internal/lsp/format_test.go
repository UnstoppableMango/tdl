package lsp_test

import (
	"path/filepath"
	"strings"
	"testing"

	"go.lsp.dev/protocol"
)

// TestFormatting replaces the document with what `tdl fmt` prints.
func TestFormatting(t *testing.T) {
	const src = "package p\ntype User: Entity { id: string   email: string }\n"
	const want = "package p\n\ntype User: Entity {\n  id: string\n  email: string\n}\n"

	s := newSession(t)
	path := abs(t, "format.tdl")
	s.open(path, src)

	edits := s.format(path)
	if len(edits) != 1 {
		t.Fatalf("edits = %d, want 1", len(edits))
	}
	if edits[0].NewText != want {
		t.Errorf("new text =\n%s\nwant\n%s", edits[0].NewText, want)
	}
	wantRange := protocol.Range{End: protocol.Position{Line: 2}}
	if edits[0].Range != wantRange {
		t.Errorf("range = %v, want the whole document %v", edits[0].Range, wantRange)
	}
}

// TestFormattingLeavesCanonicalAndBrokenFilesAlone covers the two answers
// that change nothing: text already canonical, and text that does not
// parse, which the formatter would truncate to what the parser recovered.
func TestFormattingLeavesCanonicalAndBrokenFilesAlone(t *testing.T) {
	s := newSession(t)

	canonical := abs(t, "canonical.tdl")
	s.open(canonical, "package p\n\ntype User: Entity {\n  id: string\n}\n")
	if edits := s.format(canonical); len(edits) != 0 {
		t.Errorf("canonical file edited: %v", edits)
	}

	broken := abs(t, "broken.tdl")
	s.open(broken, "package p\n\ntype User: Entity {\n  id string\n}\n")
	if edits := s.format(broken); len(edits) != 0 {
		t.Errorf("broken file edited: %v", edits)
	}
}

// TestFormattingAgreesWithTheCorpus holds the server to what
// TestCorpusIsCanonical holds `tdl fmt` to: every conformance case is
// canonical, so formatting one changes nothing.
func TestFormattingAgreesWithTheCorpus(t *testing.T) {
	for _, dir := range subdirs(t, "../../testdata/conformance") {
		t.Run(filepath.Base(dir), func(t *testing.T) {
			path, src := corpusFile(t, dir)
			s := newSession(t)
			s.open(path, src)
			if edits := s.format(path); len(edits) != 0 {
				t.Errorf("formatting a canonical file produced %d edit(s)", len(edits))
			}
		})
	}
}

func TestDocumentSymbols(t *testing.T) {
	const src = "package p\n\n" +
		"primitive string\n\n" +
		"type Email: string\n\n" +
		"deprecated\n" +
		"type User: Entity {\n" +
		"  id: string\n" +
		"  email: Email?\n" +
		"}\n\n" +
		"enum Payment {\n" +
		"  Card {\n" +
		"    last4: string\n" +
		"  }\n" +
		"  Cash\n" +
		"}\n"

	s := newSession(t)
	path := abs(t, "symbols.tdl")
	s.open(path, src)
	syms := s.symbols(path)

	got := outline(syms, "")
	want := strings.Join([]string{
		"string Class primitive",
		"Email Class string",
		"User Struct type deprecated",
		"  id Field string",
		"  email Field Email?",
		"Payment Enum",
		"  Card EnumMember",
		"    last4 Field string",
		"  Cash EnumMember",
	}, "\n") + "\n"
	if got != want {
		t.Errorf("outline =\n%s\nwant\n%s", got, want)
	}

	// The selection is the name, inside a range covering the whole body.
	user := syms[2]
	if user.SelectionRange.Start != (protocol.Position{Line: 7, Character: 5}) {
		t.Errorf("User selection starts at %v, want 7:5", user.SelectionRange.Start)
	}
	if user.Range.Start.Line != 7 || user.Range.End.Line != 10 {
		t.Errorf("User range = %v, want lines 7 to 10", user.Range)
	}
}

// outline renders symbols one per line, indented by depth, for a test to
// compare in one piece.
func outline(syms []protocol.DocumentSymbol, indent string) string {
	var b strings.Builder
	for _, sym := range syms {
		line := indent + sym.Name + " " + kindName(sym.Kind)
		if sym.Detail != nil {
			line += " " + *sym.Detail
		}
		if len(sym.Tags) > 0 {
			line += " deprecated"
		}
		b.WriteString(line + "\n")
		b.WriteString(outline(sym.Children, indent+"  "))
	}
	return b.String()
}

func kindName(k protocol.SymbolKind) string {
	return map[protocol.SymbolKind]string{
		protocol.SymbolKindClass:      "Class",
		protocol.SymbolKindStruct:     "Struct",
		protocol.SymbolKindEnum:       "Enum",
		protocol.SymbolKindEnumMember: "EnumMember",
		protocol.SymbolKindField:      "Field",
	}[k]
}

// A symbol's range has to contain its selection, or clients reject the
// whole response. Every construct in the corpus is held to it.
func TestDocumentSymbolRangesContainSelections(t *testing.T) {
	for _, dir := range subdirs(t, "../../testdata/conformance") {
		t.Run(filepath.Base(dir), func(t *testing.T) {
			path, src := corpusFile(t, dir)
			s := newSession(t)
			s.open(path, src)
			var check func(syms []protocol.DocumentSymbol)
			check = func(syms []protocol.DocumentSymbol) {
				for _, sym := range syms {
					if !contains(sym.Range, sym.SelectionRange) {
						t.Errorf("%s: range %v does not contain selection %v", sym.Name, sym.Range, sym.SelectionRange)
					}
					check(sym.Children)
				}
			}
			check(s.symbols(path))
		})
	}
}

func contains(outer, inner protocol.Range) bool {
	le := func(a, b protocol.Position) bool {
		return a.Line < b.Line || a.Line == b.Line && a.Character <= b.Character
	}
	return le(outer.Start, inner.Start) && le(inner.End, outer.End)
}
