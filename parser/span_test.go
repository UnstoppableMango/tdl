package parser_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/unstoppablemango/tdl/ast"
	"github.com/unstoppablemango/tdl/parser"
)

// TestNodesSpanTheirText checks a sample of nodes covers exactly the text
// that was written for it.
func TestNodesSpanTheirText(t *testing.T) {
	src := `package shop

import "money.tdl" as money

/// A unit of mass.
unit kg
unit N = kg*m/s^2

alias Ids = [Id]

type Email: string where { matches(/^[^@]+@[^@]+$/) length(3..254) }

deprecated("use Order") type Cart: Entity {
  id: Id
  lines: {string -> Line}? | null
  note: string = "none"
  include Audit
}

enum Status {
  Open
  Closed { at: Time }
}

class Store<a> | a -> a {
  type Cursor
}

instance Store for Cart {
  type Cursor = int
}

target go for shop {
  package("example.com/shop")
  Cart => key(id)
  Status { name("State") }
}
`
	file := parse(t, src)

	var got []string
	ast.Inspect(file, func(n ast.Node) bool {
		if n == nil {
			return false
		}
		if _, ok := n.(*ast.File); ok {
			return true
		}
		got = append(got, fmt.Sprintf("%T %s", n, src[n.Pos().Offset:n.End().Offset]))
		return true
	})

	for _, want := range []string{
		`*ast.PackageDecl package shop`,
		`*ast.ImportDecl import "money.tdl" as money`,
		`*ast.UnitDecl unit kg`,
		`*ast.UnitExpr kg*m/s^2`,
		`*ast.UnitTerm s^2`,
		`*ast.AliasDecl alias Ids = [Id]`,
		`*ast.TypeRef [Id]`,
		`*ast.NewtypeDecl type Email: string where { matches(/^[^@]+@[^@]+$/) length(3..254) }`,
		`*ast.Constraint matches(/^[^@]+@[^@]+$/)`,
		`*ast.Literal /^[^@]+@[^@]+$/`,
		`*ast.Literal 3..254`,
		`*ast.Deprecation deprecated("use Order")`,
		`*ast.ClassRef Entity`,
		`*ast.Field lines: {string -> Line}? | null`,
		`*ast.TypeRef {string -> Line}? | null`,
		`*ast.Field note: string = "none"`,
		`*ast.Literal "none"`,
		`*ast.Include include Audit`,
		`*ast.Variant Closed { at: Time }`,
		`*ast.FunDep a -> a`,
		`*ast.AssocTypeReq type Cursor`,
		`*ast.AssocTypeBind type Cursor = int`,
		`*ast.TargetEntry package("example.com/shop")`,
		`*ast.TargetEntry Cart => key(id)`,
		`*ast.Directive key(id)`,
		`*ast.TargetEntry Status { name("State") }`,
	} {
		if !contains(got, want) {
			t.Errorf("no node spans %q; got:\n%s", want, strings.Join(got, "\n"))
		}
	}
}

// TestCorpusSpansNest checks every node in the conformance corpus starts
// before it ends, ends on a token rather than space, and lies within its
// parent. A deprecation is written before the node it marks, so only its
// end is held to the parent.
func TestCorpusSpansNest(t *testing.T) {
	for _, dir := range subdirs(t, "../testdata/conformance") {
		t.Run(filepath.Base(dir), func(t *testing.T) {
			skipPending(t, dir)
			data, err := os.ReadFile(filepath.Join(dir, "source.tdl"))
			if err != nil {
				t.Fatalf("reading source.tdl: %v", err)
			}
			src := string(data)
			file, err := parser.Parse("source.tdl", strings.NewReader(src))
			if err != nil {
				t.Fatalf("unexpected parse error: %v", err)
			}

			var stack []ast.Node
			ast.Inspect(file, func(n ast.Node) bool {
				if n == nil {
					stack = stack[:len(stack)-1]
					return false
				}
				start, end := n.Pos(), n.End()
				text := src[start.Offset:end.Offset]
				if _, ok := n.(*ast.File); !ok && (end.Offset <= start.Offset || strings.TrimSpace(text) != text) {
					t.Errorf("%s: %T spans %q", start, n, text)
				}
				if len(stack) > 0 {
					parent := stack[len(stack)-1]
					_, dep := n.(*ast.Deprecation)
					if !dep && start.Offset < parent.Pos().Offset || end.Offset > parent.End().Offset {
						t.Errorf("%s: %T [%d, %d) is outside its parent %T [%d, %d)", start, n,
							start.Offset, end.Offset, parent, parent.Pos().Offset, parent.End().Offset)
					}
				}
				stack = append(stack, n)
				return true
			})
		})
	}
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}
