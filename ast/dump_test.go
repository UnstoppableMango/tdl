package ast_test

import (
	"strings"
	"testing"

	"github.com/unstoppablemango/tdl/ast"
)

func TestDump(t *testing.T) {
	file := mustParse(t, `package p

import "common.tdl" as common

primitive Map: type -> type -> type

alias Pair<T> = Map<string, T>
`)

	got := ast.Dump(file)
	for _, want := range []string{
		"File test.tdl",
		"Package p",
		`Import "common.tdl" as common`,
		"Primitive Map: type -> type -> type",
		"Alias Pair",
		"Param T",
		"Target Map<string, T>",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("dump missing %q:\n%s", want, got)
		}
	}
}

func TestDumpPackageDoc(t *testing.T) {
	file := mustParse(t, `/// What this package models.
/// Second line.
package p
`)

	got := ast.Dump(file)
	want := "└── Package p  test.tdl:3:1\n    └── Doc (2 lines)  test.tdl:1:1\n"
	if !strings.HasSuffix(got, want) {
		t.Errorf("dump does not end with %q:\n%s", want, got)
	}
}

func TestDumpWithoutPackageDoc(t *testing.T) {
	got := ast.Dump(mustParse(t, "package p\n"))
	if strings.Contains(got, "Doc") {
		t.Errorf("dump of an undocumented package has a Doc line:\n%s", got)
	}
}
