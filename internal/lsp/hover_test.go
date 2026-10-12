package lsp_test

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestHover(t *testing.T) {
	const src = "package p\n\n" +
		"primitive string\n\n" +
		"/// An address to write to.\n" +
		"type Email: string\n\n" +
		"deprecated(\"use User\")\n" +
		"type Person {\n" +
		"  name: string\n" +
		"}\n\n" +
		"type Box<T> {\n" +
		"  item: T\n" +
		"}\n\n" +
		"type User: Entity {\n" +
		"  id: string\n" +
		"  email: Email\n" +
		"  was: Person\n" +
		"  nope: Nope\n" +
		"}\n"

	s := newSession(t)
	path := abs(t, "hover.tdl")
	s.open(path, src)

	cases := []struct {
		name, needle string
		want         []string
	}{
		{"a reference", "Email\n  was", []string{"```tdl\ntype Email: string\n```", "An address to write to."}},
		{"a deprecation", "Person\n  nope", []string{"```tdl\ntype Person {\n  name: string\n}\n```", "**Deprecated**: use User"}},
		{"the prelude", "Entity", []string{"```tdl\nclass Entity"}},
		{"a declaration's own name", "User: Entity", []string{"```tdl\ntype User: Entity {\n  id: string\n"}},
		{"a type parameter", "T\n}", []string{"```tdl\nT\n```", "Type parameter."}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := s.hover(path, src, c.needle)
			for _, w := range c.want {
				if !strings.Contains(got, w) {
					t.Errorf("hover missing %q:\n%s", w, got)
				}
			}
		})
	}

	for name, needle := range map[string]string{
		"an unknown name": "Nope",
		"a keyword":       "type User",
	} {
		t.Run(name, func(t *testing.T) {
			if got := s.hover(path, src, needle); got != "" {
				t.Errorf("expected no hover, got:\n%s", got)
			}
		})
	}
}

// TestHoverCrossesAnImport describes a declaration in a file that is not
// open.
func TestHoverCrossesAnImport(t *testing.T) {
	dir := t.TempDir()
	dep := filepath.Join(dir, "common.tdl")
	writeFile(t, dep, "package common\n\nprimitive string\n\n/// An amount of money.\ntype Money {\n  amount: string\n}\n")

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
	s.open(path, src)
	got := s.hover(path, src, "Money\n}")
	for _, w := range []string{"```tdl\ntype Money {\n  amount: string\n}\n```", "An amount of money."} {
		if !strings.Contains(got, w) {
			t.Errorf("hover missing %q:\n%s", w, got)
		}
	}
}

// TestHoverListsTargetDirectives shows, where a declaration, field, or
// variant is written, what every target block resolves onto it.
func TestHoverListsTargetDirectives(t *testing.T) {
	const src = "package p\n\n" +
		"primitive string\n\n" +
		"class Named {\n" +
		"  name: string\n" +
		"}\n\n" +
		"type Commit: Named {\n" +
		"  /// A one-line summary.\n" +
		"  summary: string\n" +
		"  name: string\n" +
		"  body: string\n" +
		"}\n\n" +
		"enum Status {\n" +
		"  Open\n" +
		"  Closed { reason: string }\n" +
		"}\n\n" +
		"target protobuf for p {\n" +
		"  Commit {\n" +
		"    reserved(2, 3)\n" +
		"    summary => number(50)\n" +
		"  }\n" +
		"  Status.Closed.reason => number(4)\n" +
		"}\n\n" +
		"target go for p {\n" +
		"  Commit.summary => tag(\"json:\\\"summary\\\"\")\n" +
		"  Named.name => tag(\"json:\\\"name\\\"\")\n" +
		"  Status.Open => name(\"Opened\")\n" +
		"}\n"

	s := newSession(t)
	path := abs(t, "targets.tdl")
	s.open(path, src)

	cases := []struct {
		name, needle string
		want         []string
	}{
		{"a declaration", "Commit: Named", []string{"**protobuf**: `reserved(2, 3)`"}},
		{"a field", "summary: string", []string{
			"```tdl\nsummary: string\n```",
			"A one-line summary.",
			"**protobuf**: `number(50)`",
			"**go**: `tag(\"json:\\\"summary\\\"\")`",
		}},
		{"a class's directive", "name: string\n  body", []string{"**go**: `tag(\"json:\\\"name\\\"\")` (via Named)"}},
		{"a variant", "Open\n", []string{"```tdl\nOpen\n```", "**go**: `name(\"Opened\")`"}},
		{"a variant's field", "reason", []string{"**protobuf**: `number(4)`"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := s.hover(path, src, c.needle)
			for _, w := range c.want {
				if !strings.Contains(got, w) {
					t.Errorf("hover missing %q:\n%s", w, got)
				}
			}
		})
	}

	if got := s.hover(path, src, "body"); strings.Contains(got, "---") {
		t.Errorf("a field with no directives lists targets:\n%s", got)
	}
}
