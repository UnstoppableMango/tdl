package parser_test

import (
	"testing"

	"github.com/unstoppablemango/tdl/ast"
)

// Ordinary comments are collected on the file rather than attached to a
// node: one can sit anywhere, so the formatter places each by position.
func TestCommentsAreRecordedInSourceOrder(t *testing.T) {
	file := parse(t, `// first
primitive string // second
// third
entity E {
  // fourth
  key id: string
}
// last
`)

	want := []string{"first", "second", "third", "fourth", "last"}
	if len(file.Comments) != len(want) {
		t.Fatalf("got %d comments, want %d: %s", len(file.Comments), len(want), commentTexts(file))
	}
	for i, c := range file.Comments {
		if c.Text != want[i] {
			t.Errorf("comment %d = %q, want %q", i, c.Text, want[i])
		}
	}

	lines := []int{1, 2, 3, 5, 8}
	for i, c := range file.Comments {
		if c.P.Line != lines[i] {
			t.Errorf("comment %d on line %d, want %d", i, c.P.Line, lines[i])
		}
	}
}

// A doc comment belongs to its declaration and must not turn up in the
// file's ordinary comments as well.
func TestDocCommentsAreNotOrdinaryComments(t *testing.T) {
	file := parse(t, `/// docs
primitive string
`)

	if len(file.Comments) != 0 {
		t.Errorf("doc comment leaked into Comments: %s", commentTexts(file))
	}
	if got := ast.Doc(file.Decls[0]); len(got) != 1 || got[0] != "docs" {
		t.Errorf("doc = %v, want [docs]", got)
	}
}

// A regex literal is scanned by rewinding the lexer, which can reach a
// comment it has already passed. Recording it twice would print it twice.
func TestCommentNearRegexIsRecordedOnce(t *testing.T) {
	file := parse(t, `primitive string

// before
type Email: string where {
  matches(/^[^@]+@[^@]+$/) // after
}
`)

	want := []string{"before", "after"}
	if len(file.Comments) != len(want) {
		t.Fatalf("got %d comments, want %d: %s", len(file.Comments), len(want), commentTexts(file))
	}
	for i, c := range file.Comments {
		if c.Text != want[i] {
			t.Errorf("comment %d = %q, want %q", i, c.Text, want[i])
		}
	}
}

// A comment after the last declaration is placed against the end of the
// file, so the end has to sit past every comment. Offsets say that
// exactly, where lines do not: the lexer stops at the final newline
// without counting it.
func TestFileEndIsPastEveryComment(t *testing.T) {
	src := "primitive string\n// last\n"
	file := parse(t, src)

	if got, want := file.End.Offset, len(src); got != want {
		t.Errorf("file end at offset %d, want %d", got, want)
	}
	if len(file.Comments) != 1 {
		t.Fatalf("got %d comments, want 1", len(file.Comments))
	}
	if got := file.Comments[0].P.Offset; got >= file.End.Offset {
		t.Errorf("comment at offset %d is not before the end of the file at %d", got, file.End.Offset)
	}
}

// Every block records where it closed, which is what a comment on the last
// line inside it is placed against.
func TestBlockEndPositions(t *testing.T) {
	file := parse(t, `primitive string

entity E {
  key id: string
  n: string where {
    length(1..10)
  }
}

enum Color {
  Red
  Custom { hex: string }
}

target go for p {
  E {
    name("Entity")
  }
}
`)

	e, ok := file.Decls[1].(*ast.StructDecl)
	if !ok {
		t.Fatalf("decl 1 is %T, want *ast.StructDecl", file.Decls[1])
	}
	if got, want := e.End.Line, 8; got != want {
		t.Errorf("entity body closed on line %d, want %d", got, want)
	}

	field, ok := e.Members[1].(*ast.Field)
	if !ok {
		t.Fatalf("member 1 is %T, want *ast.Field", e.Members[1])
	}
	if got, want := field.End.Line, 7; got != want {
		t.Errorf("constraint block closed on line %d, want %d", got, want)
	}

	enum, ok := file.Decls[2].(*ast.EnumDecl)
	if !ok {
		t.Fatalf("decl 2 is %T, want *ast.EnumDecl", file.Decls[2])
	}
	if got, want := enum.End.Line, 13; got != want {
		t.Errorf("enum body closed on line %d, want %d", got, want)
	}
	if got, want := enum.Variants[1].End.Line, 12; got != want {
		t.Errorf("variant payload closed on line %d, want %d", got, want)
	}

	target, ok := file.Decls[3].(*ast.TargetDecl)
	if !ok {
		t.Fatalf("decl 3 is %T, want *ast.TargetDecl", file.Decls[3])
	}
	if got, want := target.End.Line, 19; got != want {
		t.Errorf("target block closed on line %d, want %d", got, want)
	}
	if got, want := target.Entries[0].End.Line, 18; got != want {
		t.Errorf("nested target block closed on line %d, want %d", got, want)
	}
}

// A field without a constraint block has no block to record.
func TestFieldWithoutConstraintsHasNoEnd(t *testing.T) {
	file := parse(t, "primitive string\n\nentity E {\n  key id: string\n}\n")

	e := file.Decls[1].(*ast.StructDecl)
	if got := e.Members[0].(*ast.Field).End; got.Line != 0 {
		t.Errorf("field end = %v, want the zero position", got)
	}
}

func commentTexts(file *ast.File) string {
	var s string
	for _, c := range file.Comments {
		s += " " + c.Text
	}
	return s
}
