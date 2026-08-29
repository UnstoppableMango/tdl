package parser_test

import (
	"strings"
	"testing"

	"github.com/unstoppablemango/tdl/ast"
	"github.com/unstoppablemango/tdl/parser"
)

func parse(t *testing.T, src string) *ast.File {
	t.Helper()
	file, err := parser.Parse("test.tdl", strings.NewReader(src))
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}
	return file
}

func parseErr(t *testing.T, src string) string {
	t.Helper()
	_, err := parser.Parse("test.tdl", strings.NewReader(src))
	if err == nil {
		t.Fatal("expected a parse error, got none")
	}
	return err.Error()
}

func TestPackageAndImports(t *testing.T) {
	file := parse(t, `
package shop.orders

import "common.tdl" as common
import "std/prelude" as _
`)

	if file.Package == nil || file.Package.Path != "shop.orders" {
		t.Fatalf("package = %+v, want shop.orders", file.Package)
	}
	if len(file.Imports) != 2 {
		t.Fatalf("got %d imports, want 2", len(file.Imports))
	}
	if got := file.Imports[1].Alias; got != "_" {
		t.Errorf("second import alias = %q, want _", got)
	}
}

func TestPrimitiveKinds(t *testing.T) {
	file := parse(t, `
primitive string
primitive Map: type -> type -> type
primitive Higher: (type -> type) -> type
primitive Measured: unit -> type
`)

	if len(file.Decls) != 4 {
		t.Fatalf("got %d decls, want 4", len(file.Decls))
	}

	if p := file.Decls[0].(*ast.PrimitiveDecl); p.Kind != nil {
		t.Errorf("string kind = %+v, want nil", p.Kind)
	}

	// `type -> type -> type` associates to the right.
	m := file.Decls[1].(*ast.PrimitiveDecl)
	if m.Kind.N != "type" || m.Kind.Arrow == nil || m.Kind.Arrow.Arrow == nil {
		t.Fatalf("Map kind did not associate right: %+v", m.Kind)
	}
	if m.Kind.Arrow.Arrow.Arrow != nil {
		t.Error("Map kind has a fourth atom")
	}

	// A parenthesized kind is the left operand, not the right.
	h := file.Decls[2].(*ast.PrimitiveDecl)
	if h.Kind.Paren == nil {
		t.Fatalf("Higher kind lost its parentheses: %+v", h.Kind)
	}
	if h.Kind.Paren.Arrow == nil || h.Kind.Arrow == nil {
		t.Errorf("Higher kind = %+v", h.Kind)
	}
}

func TestAliasParameters(t *testing.T) {
	file := parse(t, `alias Applied<f: type -> type, T> = f<T>`)

	a := file.Decls[0].(*ast.AliasDecl)
	if len(a.Params) != 2 {
		t.Fatalf("got %d params, want 2", len(a.Params))
	}
	if a.Params[0].Kind == nil {
		t.Error("first param lost its kind annotation")
	}
	if a.Params[1].Kind != nil {
		t.Error("second param gained a kind annotation")
	}
	if a.Target.N != "f" || len(a.Target.Args) != 1 {
		t.Errorf("target = %+v", a.Target)
	}
}

func TestTypeRefForms(t *testing.T) {
	tests := []struct {
		src   string
		check func(*testing.T, *ast.TypeRef)
	}{
		{"[T]", func(t *testing.T, r *ast.TypeRef) {
			if r.List == nil || r.List.N != "T" {
				t.Errorf("list = %+v", r)
			}
		}},
		{"{T}", func(t *testing.T, r *ast.TypeRef) {
			if r.Set == nil || r.Set.N != "T" {
				t.Errorf("set = %+v", r)
			}
		}},
		{"{K -> V}", func(t *testing.T, r *ast.TypeRef) {
			if r.MapKey == nil || r.MapValue == nil {
				t.Errorf("map = %+v", r)
			}
		}},
		{"T?", func(t *testing.T, r *ast.TypeRef) {
			if !r.Optional || r.Nullable {
				t.Errorf("optional = %+v", r)
			}
		}},
		{"T | null", func(t *testing.T, r *ast.TypeRef) {
			if r.Optional || !r.Nullable {
				t.Errorf("nullable = %+v", r)
			}
		}},
		{"T? | null", func(t *testing.T, r *ast.TypeRef) {
			if !r.Optional || !r.Nullable {
				t.Errorf("both = %+v", r)
			}
		}},
		{"common.Address", func(t *testing.T, r *ast.TypeRef) {
			if r.Qualifier != "common" || r.N != "Address" {
				t.Errorf("qualified = %+v", r)
			}
		}},
		{"Map<string, [T]>", func(t *testing.T, r *ast.TypeRef) {
			if len(r.Args) != 2 || r.Args[1].List == nil {
				t.Errorf("args = %+v", r)
			}
		}},
	}

	for _, tt := range tests {
		t.Run(tt.src, func(t *testing.T) {
			file := parse(t, "alias X = "+tt.src)
			tt.check(t, file.Decls[0].(*ast.AliasDecl).Target)
		})
	}
}

func TestDocComments(t *testing.T) {
	file := parse(t, `
/// A handler table.
/// Keyed by event name.
alias Handler = {string -> [Event]}
`)

	doc := ast.Doc(file.Decls[0])
	if len(doc) != 2 || doc[0] != "A handler table." {
		t.Errorf("doc = %q", doc)
	}
}

// Whitespace is insignificant: a declaration ends where the next begins.
func TestWhitespaceInsignificant(t *testing.T) {
	oneLine := parse(t, `primitive string primitive int alias A = string`)
	if len(oneLine.Decls) != 3 {
		t.Fatalf("got %d decls on one line, want 3", len(oneLine.Decls))
	}
}

func TestErrorsAccumulate(t *testing.T) {
	msg := parseErr(t, `
import "a.tdl"
alias Broken =
`)
	if !strings.Contains(msg, "expected as") {
		t.Errorf("error missing the import problem: %s", msg)
	}
}

func TestReservedKeywordsCannotName(t *testing.T) {
	if msg := parseErr(t, `alias entity = string`); !strings.Contains(msg, "expected identifier") {
		t.Errorf("error = %s", msg)
	}
}

// Modifier and constraint names are contextual, so they stay usable as
// ordinary identifiers.
func TestContextualKeywordsAreIdentifiers(t *testing.T) {
	for _, name := range []string{"key", "owned", "deprecated", "min", "max", "length", "matches", "oneOf", "unique"} {
		t.Run(name, func(t *testing.T) {
			file := parse(t, "alias "+name+" = string")
			if got := file.Decls[0].Name(); got != name {
				t.Errorf("name = %q, want %q", got, name)
			}
		})
	}
}
