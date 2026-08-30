package ast_test

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

func TestFprintGolden(t *testing.T) {
	src := `package example.v1

import "common.tdl" as common

@go(pkg: "example")
type User {
  id: string
  name: string?
  tags: list<string>
  metadata: map<string, string>
  address: common.Address?
  role: Role = "member"
}

enum Role {
  Admin = "admin"
  Member = "member"
  Guest = "guest"
}
`
	got := ast.Fprint(parse(t, src))
	if got != src {
		t.Fatalf("Fprint mismatch:\n--- got ---\n%s\n--- want ---\n%s", got, src)
	}
}

func TestFprintIsIdempotent(t *testing.T) {
	src := `
type   User    {
id:string
  name : string ?
}
`
	once := ast.Fprint(parse(t, src))
	twice := ast.Fprint(parse(t, once))
	if once != twice {
		t.Fatalf("formatting is not idempotent:\n--- once ---\n%s\n--- twice ---\n%s", once, twice)
	}
}
