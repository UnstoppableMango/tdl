package protobuf_test

import (
	"strings"
	"testing"
)

// An option in the target block's scope is a file option, written after the
// package and the imports and before the first declaration.
func TestFileOptionIsWritten(t *testing.T) {
	const src = `package shop

type Account { id: string }

target protobuf for shop {
  import("google/api/field_behavior.proto")
  option("deprecated", "true")
  option("java_package", "\"com.acme.shop\"")
}
`
	out := compileWith(t, generateModel(t, src), googleAPI)
	contains(t, out,
		`package shop;

import "google/api/field_behavior.proto";

option deprecated = true;
option java_package = "com.acme.shop";

message Account { string id = 1; }`,
	)
}

// Every file a `file` directive splits out carries the block's file options,
// as it carries the block's imports.
func TestFileOptionInEveryFile(t *testing.T) {
	const src = `package acme.cli.v1

type Token { text: string }

type Command { tokens: List<Token> }

target protobuf for acme.cli.v1 {
  option("java_package", "\"com.acme.cli.v1\"")
  Token { file("cst.proto") }
}
`
	files := compileAll(t, generateModel(t, src))
	if len(files) != 2 {
		t.Fatalf("files = %d, want 2: %v", len(files), files)
	}
	for path, out := range files {
		if !strings.Contains(out, `option java_package = "com.acme.cli.v1";`) {
			t.Errorf("%s does not carry the file option:\n%s", path, out)
		}
	}
}
