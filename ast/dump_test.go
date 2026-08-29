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
