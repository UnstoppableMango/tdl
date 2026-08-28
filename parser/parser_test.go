package parser_test

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

func TestParsePackageAndImport(t *testing.T) {
	file := mustParse(t, `
package example.v1

import "common.tdl" as common
`)
	if file.Package == nil || file.Package.Name != "example.v1" {
		t.Fatalf("got package %+v, want example.v1", file.Package)
	}
	if len(file.Imports) != 1 || file.Imports[0].Path != "common.tdl" || file.Imports[0].Alias != "common" {
		t.Fatalf("got imports %+v", file.Imports)
	}
}

func TestParseRecord(t *testing.T) {
	file := mustParse(t, `
type User {
  id: string
  name: string?
  tags: list<string>
  metadata: map<string, string>
  address: common.Address?
  role: Role = "member"
}
`)
	if len(file.Types) != 1 {
		t.Fatalf("got %d types, want 1", len(file.Types))
	}
	ty := file.Types[0]
	if ty.Name != "User" {
		t.Fatalf("got name %q, want User", ty.Name)
	}
	if len(ty.Fields) != 6 {
		t.Fatalf("got %d fields, want 6", len(ty.Fields))
	}

	id := ty.Fields[0]
	if id.Name != "id" || id.Type.Name != "string" || id.Optional {
		t.Fatalf("id field: %+v", id)
	}

	name := ty.Fields[1]
	if name.Name != "name" || !name.Optional {
		t.Fatalf("name field: %+v", name)
	}

	tags := ty.Fields[2]
	if tags.Type.List == nil || tags.Type.List.Name != "string" {
		t.Fatalf("tags field type: %+v", tags.Type)
	}

	metadata := ty.Fields[3]
	if metadata.Type.MapKey == nil || metadata.Type.MapKey.Name != "string" ||
		metadata.Type.MapValue == nil || metadata.Type.MapValue.Name != "string" {
		t.Fatalf("metadata field type: %+v", metadata.Type)
	}

	address := ty.Fields[4]
	if address.Type.Qualifier != "common" || address.Type.Name != "Address" || !address.Optional {
		t.Fatalf("address field: %+v", address)
	}

	role := ty.Fields[5]
	if role.Default == nil || role.Default.Kind != ast.LitString || role.Default.Str != "member" {
		t.Fatalf("role field default: %+v", role.Default)
	}
}

func TestParseEnum(t *testing.T) {
	file := mustParse(t, `
enum Role {
  Admin = "admin"
  Member = "member"
  Guest = "guest"
}
`)
	if len(file.Enums) != 1 {
		t.Fatalf("got %d enums, want 1", len(file.Enums))
	}
	e := file.Enums[0]
	if e.Name != "Role" || len(e.Values) != 3 {
		t.Fatalf("got %+v", e)
	}
	if e.Values[0].Name != "Admin" || e.Values[0].Value.Str != "admin" {
		t.Fatalf("got %+v", e.Values[0])
	}
}

func TestParseAnnotations(t *testing.T) {
	file := mustParse(t, `
@go(pkg: "example")
type User {
  @go(tag: "json:\"full_name,omitempty\"")
  @protobuf(number: 10)
  name: string
}
`)
	ty := file.Types[0]
	if len(ty.Annotations) != 1 || ty.Annotations[0].Namespace != "go" {
		t.Fatalf("type annotations: %+v", ty.Annotations)
	}
	if ty.Annotations[0].Args[0].Name != "pkg" || ty.Annotations[0].Args[0].Value.Str != "example" {
		t.Fatalf("type annotation arg: %+v", ty.Annotations[0].Args[0])
	}

	field := ty.Fields[0]
	if len(field.Annotations) != 2 {
		t.Fatalf("got %d field annotations, want 2", len(field.Annotations))
	}
	if field.Annotations[1].Namespace != "protobuf" || field.Annotations[1].Args[0].Value.Int != 10 {
		t.Fatalf("protobuf annotation: %+v", field.Annotations[1])
	}
}

func TestParseAnnotationListValue(t *testing.T) {
	file := mustParse(t, `
@validate(oneOf: ["a", "b", "c"])
type T {
  f: string
}
`)
	arg := file.Types[0].Annotations[0].Args[0]
	if arg.Value.Kind != ast.LitList || len(arg.Value.List) != 3 || arg.Value.List[1].Str != "b" {
		t.Fatalf("got %+v", arg.Value)
	}
}

func TestParseErrorsCollectMultiple(t *testing.T) {
	_, err := parser.Parse("test.tdl", strings.NewReader(`
type User {
  id string
  name:
}

enum {
}
`))
	if err == nil {
		t.Fatal("expected a parse error")
	}
	errs, ok := err.(parser.ErrorList)
	if !ok {
		t.Fatalf("expected parser.ErrorList, got %T", err)
	}
	if len(errs) < 2 {
		t.Fatalf("expected multiple collected errors, got %d: %v", len(errs), errs)
	}
}

func TestParseUnterminatedType(t *testing.T) {
	_, err := parser.Parse("test.tdl", strings.NewReader(`type User {`))
	if err == nil {
		t.Fatal("expected a parse error")
	}
}
