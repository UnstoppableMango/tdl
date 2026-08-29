package ast_test

import (
	"strings"
	"testing"

	"github.com/unstoppablemango/tdl/ast"
	"github.com/unstoppablemango/tdl/parser"
)

func mustParse(t *testing.T, src string) *ast.File {
	t.Helper()
	file, err := parser.Parse("test.tdl", strings.NewReader(src))
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}
	return file
}

func TestFprintCanonical(t *testing.T) {
	src := `package example.aliases

import "common.tdl" as common

/// A handler table.
primitive string

primitive Map: type -> type -> type

primitive Higher: (type -> type) -> type

alias Applied<f: type -> type, T> = f<T>

alias Handler = {string -> [Event]}

alias Both = LineItem? | null

alias Qualified = Map<string, common.Address>
`

	got := ast.Fprint(mustParse(t, src))
	if got != src {
		t.Errorf("Fprint mismatch\n--- got ---\n%s\n--- want ---\n%s", got, src)
	}
}

// Formatting canonical output must change nothing, and formatting anything
// else must reach canonical output in one pass.
func TestFprintIdempotent(t *testing.T) {
	messy := `package   p
primitive string primitive int
alias   A=[  T ]
alias B = { K->V }
`

	once := ast.Fprint(mustParse(t, messy))
	twice := ast.Fprint(mustParse(t, once))
	if once != twice {
		t.Errorf("not idempotent\n--- once ---\n%s\n--- twice ---\n%s", once, twice)
	}
}
