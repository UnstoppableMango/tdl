package ast_test

import (
	"strings"
	"testing"

	"github.com/unstoppablemango/tdl/ast"
)

// An ordinary comment survives formatting, wherever it was written. A
// comment attaches to no node, so this is the property the position-driven
// placement exists to hold.
func TestFprintKeepsComments(t *testing.T) {
	src := `// before the package
package p

// before the import
import "common.tdl" as common

primitive string  // trailing a declaration

// before the entity
type E: Entity {  // opening a body
  // before a field
  id: string  // trailing a field
  n: string where {
    // inside a where block
    min(0)  // trailing a constraint
  }
  // last inside the body
}

enum Color {  // opening an enum
  Red
  // between variants
  Blue
}

target go for p {
  // inside a target block
  out("./gen")  // trailing a directive
  E {
    // inside a nested block
    name("Entity")
  }
}

// after the last declaration
`

	got := ast.Fprint(mustParse(t, src))
	if got != src {
		t.Errorf("Fprint mismatch\n--- got ---\n%s\n--- want ---\n%s", got, src)
	}
}

// Every comment that went in comes back out, whatever the input layout.
func TestFprintDropsNoComment(t *testing.T) {
	src := `package p
primitive string
// one
type E: Entity{// two
id:string// three
n:string where{// four
min(0)}// five
}// six
// seven
`

	got := ast.Fprint(mustParse(t, src))
	for _, want := range []string{"one", "two", "three", "four", "five", "six", "seven"} {
		if !strings.Contains(got, "// "+want) {
			t.Errorf("comment %q was dropped:\n%s", want, got)
		}
	}
}

// A one-line block has nowhere to put a comment, so one inside forces the
// expanded form.
func TestFprintExpandsBlocksHoldingComments(t *testing.T) {
	cases := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "enum body",
			src:  "package p\nprimitive string\nenum Color { Red Blue }\n",
			want: "enum Color { Red Blue }\n",
		},
		{
			name: "enum body with a comment",
			src:  "package p\nprimitive string\nenum Color {\n  Red\n  // why\n  Blue\n}\n",
			want: "enum Color {\n  Red\n  // why\n  Blue\n}\n",
		},
		{
			name: "single constraint",
			src:  "package p\nprimitive string\ntype T: string where { min(0) }\n",
			want: "type T: string where { min(0) }\n",
		},
		{
			name: "single constraint with a comment",
			src:  "package p\nprimitive string\ntype T: string where {\n  // why\n  min(0)\n}\n",
			want: "type T: string where {\n  // why\n  min(0)\n}\n",
		},
		{
			name: "empty body with a comment",
			src:  "package p\ntype E: Entity {\n  // nothing yet\n}\n",
			want: "type E: Entity {\n  // nothing yet\n}\n",
		},
		{
			name: "variant payload with a comment",
			src:  "package p\nprimitive string\nenum E {\n  V {\n    // why\n    n: string\n  }\n}\n",
			want: "  V {\n    // why\n    n: string\n  }\n",
		},
		{
			name: "empty variant payload with a comment",
			src:  "package p\nenum E {\n  V {\n    // nothing yet\n  }\n}\n",
			want: "  V {\n    // nothing yet\n  }\n",
		},
		{
			name: "variant payload with a field doc comment",
			src:  "package p\nprimitive string\nenum E {\n  V { /// the name\n    n: string }\n}\n",
			want: "  V {\n    /// the name\n    n: string\n  }\n",
		},
		{
			name: "variant payload with a multi-line constraint block",
			src:  "package p\nprimitive string\nenum E {\n  V { n: string where { min(1) max(9) } }\n}\n",
			want: "  V {\n    n: string where {\n      min(1)\n      max(9)\n    }\n  }\n",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ast.Fprint(mustParse(t, tc.src))
			if !strings.Contains(got, tc.want) {
				t.Errorf("output does not contain %q:\n%s", tc.want, got)
			}
		})
	}
}

// A doc comment still belongs to its declaration, and is not written twice
// now that ordinary comments are written at all.
func TestFprintDocAndOrdinaryCommentsCoexist(t *testing.T) {
	src := `package p

/// what it is
// how it got here
primitive string
`

	got := ast.Fprint(mustParse(t, src))
	if strings.Count(got, "/// what it is") != 1 {
		t.Errorf("doc comment written %d times:\n%s", strings.Count(got, "/// what it is"), got)
	}
	if strings.Count(got, "// how it got here") != 1 {
		t.Errorf("ordinary comment written %d times:\n%s", strings.Count(got, "// how it got here"), got)
	}
}

// Formatting a commented file twice must reach the same text, or a comment
// would drift a line on every run.
func TestFprintIdempotentWithComments(t *testing.T) {
	messy := `// header
package   p
primitive string// one
// two
type  E: Entity{id:string// three
  n:int where{// four
min(0) max(10)}}
enum Big { A B // five
C }
// tail
`

	once := ast.Fprint(mustParse(t, messy))
	twice := ast.Fprint(mustParse(t, once))
	if once != twice {
		t.Errorf("not idempotent\n--- once ---\n%s\n--- twice ---\n%s", once, twice)
	}
}

// A comment after a block's closing brace stays there when the formatter
// opens the block up. It shares a source line with the first item inside,
// and the item's line is not where it belongs.
func TestFprintKeepsTrailingCommentAfterBlock(t *testing.T) {
	cases := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "entity body",
			src:  "package p\nprimitive string\ntype E: Entity { a: string b: string } // c\n",
			want: "type E: Entity {\n  a: string\n  b: string\n}  // c\n",
		},
		{
			name: "constraint block",
			src:  "package p\nprimitive string\ntype E: Entity {\n  a: string where { min(1) max(9) } // c\n}\n",
			want: "  a: string where {\n    min(1)\n    max(9)\n  }  // c\n",
		},
		{
			name: "nested target block",
			src:  "package p\ntarget go for x {\n  E { a => tag(\"x\") } // c\n}\n",
			want: "  E {\n    a => tag(\"x\")\n  }  // c\n",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ast.Fprint(mustParse(t, tc.src))
			if !strings.Contains(got, tc.want) {
				t.Errorf("output does not contain %q:\n%s", tc.want, got)
			}
		})
	}
}

// A comment sharing a source line with a block's first item belongs to that
// item, not to the opening brace. The whole block is on one line until the
// formatter opens it up, so the brace and the item start out on the line
// the comment was written on, and only the item follows it.
func TestFprintBindsCommentToFirstItem(t *testing.T) {
	cases := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "entity body",
			src:  "package p\nprimitive string\ntype E: Entity { a: string // c\n}\n",
			want: "type E: Entity {\n  a: string  // c\n}\n",
		},
		{
			name: "enum body",
			src:  "package p\nenum E { A // c\n}\n",
			want: "enum E {\n  A  // c\n}\n",
		},
		{
			name: "variant body",
			src:  "package p\nprimitive string\nenum E { A { a: string // c\n} }\n",
			want: "  A {\n    a: string  // c\n  }\n",
		},
		{
			name: "instance binds",
			src:  "package p\ninstance C for T { type A = B // c\n}\n",
			want: "instance C for T {\n  type A = B  // c\n}\n",
		},
		{
			name: "target block",
			src:  "package p\ntarget go for x { out(\"./gen\") // c\n}\n",
			want: "target go for x {\n  out(\"./gen\")  // c\n}\n",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ast.Fprint(mustParse(t, tc.src))
			if !strings.Contains(got, tc.want) {
				t.Errorf("output does not contain %q:\n%s", tc.want, got)
			}
			if twice := ast.Fprint(mustParse(t, got)); twice != got {
				t.Errorf("not idempotent\n--- once ---\n%s\n--- twice ---\n%s", got, twice)
			}
		})
	}
}

// A comment written before a block's first item belongs to the brace, and
// stays folded onto it.
func TestFprintKeepsCommentOnOpeningBrace(t *testing.T) {
	src := "package p\nprimitive string\ntype E: Entity { // c\n  a: string\n}\n"
	want := "type E: Entity {  // c\n  a: string\n}\n"

	if got := ast.Fprint(mustParse(t, src)); !strings.Contains(got, want) {
		t.Errorf("output does not contain %q:\n%s", want, got)
	}
}
