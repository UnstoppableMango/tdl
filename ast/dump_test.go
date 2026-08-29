package ast_test

import (
	"testing"

	"github.com/unstoppablemango/tdl/ast"
)

func TestDumpGolden(t *testing.T) {
	src := `package example.v1

import "common.tdl" as common

@go(pkg: "example")
type User {
  id: string
  address: common.Address?

  @protobuf(number: 10)
  role: Role = "member"
}

enum Role {
  Admin = "admin"
  Guest
}
`

	want := "File test.tdl\n" +
		"|- Package example.v1  1:1\n" +
		"|- Import \"common.tdl\" as common  3:1\n" +
		"|- Type User  6:1\n" +
		"|  |- @go(pkg: \"example\")  5:1\n" +
		"|  |- Field id: string  7:3\n" +
		"|  |- Field address: common.Address?  8:3\n" +
		"|  `- Field role: Role = \"member\"  11:3\n" +
		"|     `- @protobuf(number: 10)  10:3\n" +
		"`- Enum Role  14:1\n" +
		"   |- Value Admin = \"admin\"  15:3\n" +
		"   `- Value Guest  16:3\n"

	got := ast.Dump(parse(t, src))
	if got != want {
		t.Errorf("Dump mismatch\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestDumpEmptyFile(t *testing.T) {
	got := ast.Dump(parse(t, ""))
	if got != "File test.tdl\n" {
		t.Errorf("Dump of empty file = %q", got)
	}
}
