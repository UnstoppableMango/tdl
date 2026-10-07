package ast_test

import (
	"strings"
	"testing"

	"github.com/unstoppablemango/tdl/ast"
)

// An ordinary comment survives formatting wherever it was written.
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

// A comment inside a block forces the expanded form.
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

// A doc comment is written once, beside ordinary comments.
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

// A comment after a one-line block's closing brace stays after it when
// the block opens up.
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

// In a one-line block, a comment after the first item folds onto that
// item, not the opening brace.
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

func TestFprintKeepsCommentOnOpeningBrace(t *testing.T) {
	src := "package p\nprimitive string\ntype E: Entity { // c\n  a: string\n}\n"
	want := "type E: Entity {  // c\n  a: string\n}\n"

	if got := ast.Fprint(mustParse(t, src)); !strings.Contains(got, want) {
		t.Errorf("output does not contain %q:\n%s", want, got)
	}
}

// A doc comment and an ordinary comment above the same item keep their
// order.
func TestFprintKeepsDocAndCommentOrder(t *testing.T) {
	tests := map[string]string{
		"declaration, doc first": `package p

/// what it is
// how it got here
primitive string
`,
		"declaration, comment first": `package p

// how it got here
/// what it is
primitive string
`,
		"import, doc first": `package p

/// what it is
// how it got here
import "common.tdl" as common
`,
		"import, comment first": `package p

// how it got here
/// what it is
import "common.tdl" as common
`,
		"field, doc first": `package p

type E: Entity {
  /// what it is
  // how it got here
  id: string
}
`,
		"field, comment first": `package p

type E: Entity {
  // how it got here
  /// what it is
  id: string
}
`,
		"variant, doc first": `package p

enum Color {
  /// what it is
  // how it got here
  Red
  Blue
}
`,
		"variant, comment first": `package p

enum Color {
  // how it got here
  /// what it is
  Red
  Blue
}
`,
		"associated type, doc first": `package p

class C<T> {
  /// what it is
  // how it got here
  type Cursor
}
`,
		"associated type, comment first": `package p

class C<T> {
  // how it got here
  /// what it is
  type Cursor
}
`,
		"declaration, comment inside": `package p

/// what it is
// how it got here
/// and more
primitive string
`,
		"import, comment inside": `package p

/// what it is
// how it got here
/// and more
import "common.tdl" as common
`,
		"field, comment inside": `package p

type E: Entity {
  /// what it is
  // how it got here
  /// and more
  id: string
}
`,
		"variant, comment inside": `package p

enum Color {
  /// what it is
  // how it got here
  /// and more
  Red
  Blue
}
`,
		"associated type, comment inside": `package p

class C<T> {
  /// what it is
  // how it got here
  /// and more
  type Cursor
}
`,
		"package, doc first": `/// what it is
// how it got here
package p
`,
		"package, comment first": `// how it got here
/// what it is
package p
`,
	}

	for name, src := range tests {
		t.Run(name, func(t *testing.T) {
			got := ast.Fprint(mustParse(t, src))
			if got != src {
				t.Errorf("Fprint mismatch\n--- got ---\n%s\n--- want ---\n%s", got, src)
			}
		})
	}
}

func TestFprintKeepsBlankLineBetweenCommentGroups(t *testing.T) {
	src := `package acme.v1

// What this package models.

// Widgets

type Widget {
  name: string
}
`
	want := "// What this package models.\n\n// Widgets\n"

	got := ast.Fprint(mustParse(t, src))
	if !strings.Contains(got, want) {
		t.Errorf("blank line between comment groups was dropped\n--- got ---\n%s\n--- want substring ---\n%s", got, want)
	}
}

func TestFprintKeepsBlankLineBetweenCommentGroupsAbovePackage(t *testing.T) {
	src := "// License header.\n\n// What this package models.\npackage acme.v1\n"

	if got := ast.Fprint(mustParse(t, src)); got != src {
		t.Errorf("Fprint mismatch\n--- got ---\n%s\n--- want ---\n%s", got, src)
	}
}

// Blank lines between a top-level comment and its declaration collapse to
// one; a comment directly above stays attached.
func TestFprintBlankLineBetweenCommentAndDeclaration(t *testing.T) {
	cases := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "one blank line is kept",
			src:  "package acme.v1\n\n// Widgets\n\ntype Widget {\n  name: string\n}\n",
			want: "package acme.v1\n\n// Widgets\n\ntype Widget {\n  name: string\n}\n",
		},
		{
			name: "no blank line stays attached",
			src:  "package acme.v1\n\n// Widgets\ntype Widget {\n  name: string\n}\n",
			want: "package acme.v1\n\n// Widgets\ntype Widget {\n  name: string\n}\n",
		},
		{
			name: "several blank lines collapse to one",
			src:  "package acme.v1\n\n// Widgets\n\n\n\ntype Widget {\n  name: string\n}\n",
			want: "package acme.v1\n\n// Widgets\n\ntype Widget {\n  name: string\n}\n",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ast.Fprint(mustParse(t, tc.src))
			if got != tc.want {
				t.Errorf("Fprint mismatch\n--- got ---\n%s\n--- want ---\n%s", got, tc.want)
			}
		})
	}
}
