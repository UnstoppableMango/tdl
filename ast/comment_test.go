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
entity E {  // opening a body
  // before a field
  key id: string  // trailing a field
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
entity E{// two
key id:string// three
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
			src:  "package p\nentity E {\n  // nothing yet\n}\n",
			want: "entity E {\n  // nothing yet\n}\n",
		},
		{
			name: "variant payload with a comment",
			src:  "package p\nprimitive string\nenum E {\n  V {\n    // why\n    n: string\n  }\n}\n",
			want: "  V {\n    // why\n    n: string\n  }\n",
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
entity  E{key id:string// three
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
