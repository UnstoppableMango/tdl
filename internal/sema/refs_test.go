package sema_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/unstoppablemango/tdl/internal/sema"
	"github.com/unstoppablemango/tdl/parser"
)

// lowerWithRefs lowers src and returns the reference index.
func lowerWithRefs(t *testing.T, src string) sema.References {
	t.Helper()

	file, err := parser.Parse("refs.tdl", strings.NewReader(src))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	var refs sema.References
	_, diags := sema.Lower(file, sema.WithReferences(&refs))
	if len(diags) > 0 {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	return refs
}

// at is the reference covering the first occurrence of needle in src.
func at(t *testing.T, refs sema.References, src, needle string) sema.Reference {
	t.Helper()

	off := strings.Index(src, needle)
	if off < 0 {
		t.Fatalf("%q is not in the source", needle)
	}

	ref, ok := refs.At("refs.tdl", off)
	if !ok {
		t.Fatalf("no reference at %q (offset %d)", needle, off)
	}
	return ref
}

// TestReferencesResolveToTheirDeclaration is the property go to definition
// is built on: a name records where what it names was written.
func TestReferencesResolveToTheirDeclaration(t *testing.T) {
	const src = "package p\n\n" +
		"primitive string\n\n" +
		"type Email: string\n\n" +
		"type User: Entity {\n" +
		"  id: string\n" +
		"  email: Email\n" +
		"}\n"

	refs := lowerWithRefs(t, src)

	ref := at(t, refs, src, "Email\n}")
	if ref.Name != "Email" {
		t.Fatalf("name = %q, want Email", ref.Name)
	}
	if ref.Target.Line != 5 {
		t.Errorf("target line = %d, want 5 (the `type Email` declaration)", ref.Target.Line)
	}
	if ref.Len != len("Email") {
		t.Errorf("len = %d, want %d", ref.Len, len("Email"))
	}
}

// TestReferencesCoverEveryMention is what the type table cannot answer.
//
// Types are interned by structure, so the second `string` in a file is not
// a second entry and a cursor on it would find nothing. The index records
// the question rather than the answer, so every mention is there.
func TestReferencesCoverEveryMention(t *testing.T) {
	const src = "package p\n\n" +
		"primitive string\n\n" +
		"type User: Entity {\n" +
		"  id: string\n" +
		"  name: string\n" +
		"}\n"

	refs := lowerWithRefs(t, src)

	first := strings.Index(src, "id: string") + len("id: ")
	second := strings.Index(src, "name: string") + len("name: ")

	for _, off := range []int{first, second} {
		ref, ok := refs.At("refs.tdl", off)
		if !ok {
			t.Fatalf("no reference at offset %d", off)
		}
		if ref.Name != "string" || ref.Target.Line != 3 {
			t.Errorf("at %d: got %q at line %d, want string at line 3", off, ref.Name, ref.Target.Line)
		}
	}
}

// TestReferencesFindTypeParameters checks the binding that shadows a
// declaration: inside a parameterized declaration, T is the parameter.
func TestReferencesFindTypeParameters(t *testing.T) {
	const src = "package p\n\n" +
		"primitive int\n\n" +
		"type Page<T> {\n" +
		"  items: [T]\n" +
		"  total: int\n" +
		"}\n"

	refs := lowerWithRefs(t, src)

	ref := at(t, refs, src, "T]")
	if ref.Name != "T" {
		t.Fatalf("name = %q, want T", ref.Name)
	}
	// The parameter is declared in `Page<T>` on line 5, not by a
	// declaration of its own, so the target is where it was bound.
	if ref.Target.Line != 5 {
		t.Errorf("target line = %d, want 5", ref.Target.Line)
	}
	if ref.Decl != nil {
		t.Error("a type parameter declares nothing, so Decl should be unset")
	}
}

// TestReferencesFindTargetPaths covers the mistake a model author actually
// makes: a target path naming a declaration.
func TestReferencesFindTargetPaths(t *testing.T) {
	const src = "package p\n\n" +
		"primitive string\n\n" +
		"type User: Entity {\n" +
		"  email: string\n" +
		"}\n\n" +
		"target go for p {\n" +
		"  User.email => tag(\"json:email\")\n" +
		"}\n"

	refs := lowerWithRefs(t, src)

	ref := at(t, refs, src, "User.email =>")
	if ref.Name != "User" {
		t.Fatalf("name = %q, want User", ref.Name)
	}
	if ref.Target.Line != 5 {
		t.Errorf("target line = %d, want 5", ref.Target.Line)
	}
}

// TestReferencesCrossAnImport covers the name that is declared in another
// file: a `_` import merges a dependency's exported names in, and what a
// reference to one records is where that dependency declares it.
func TestReferencesCrossAnImport(t *testing.T) {
	dir := t.TempDir()
	dep := filepath.Join(dir, "common.tdl")
	write(t, dep, "package common\n\nprimitive string\n\ntype Money {\n  amount: string\n}\n")

	src := "package p\n\n" +
		"import \"common.tdl\" as _\n\n" +
		"primitive string\n\n" +
		"type Order: Entity {\n" +
		"  id: string\n" +
		"  total: Money\n" +
		"}\n"
	main := filepath.Join(dir, "order.tdl")
	write(t, main, src)

	file, err := parser.Parse(main, strings.NewReader(src))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	var refs sema.References
	if _, diags := sema.Lower(file,
		sema.WithReferences(&refs),
		sema.WithLoader(sema.FSLoader{}),
	); len(diags) > 0 {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	ref, ok := refs.At(main, strings.Index(src, "Money"))
	if !ok {
		t.Fatal("no reference on Money")
	}
	if ref.Target.Filename != dep {
		t.Errorf("target file = %s, want %s", ref.Target.Filename, dep)
	}
	if ref.Target.Line != 5 {
		t.Errorf("target line = %d, want 5", ref.Target.Line)
	}
}

// TestQualifiedReferencesHaveNoTarget is the other half of an import: an
// aliased dependency is parsed for its names and not for its positions, so
// a qualified reference records the question and no answer. The record
// still earns its place, because it is what stops a cursor on
// `common.Money` from finding whatever reference sits next to it.
func TestQualifiedReferencesHaveNoTarget(t *testing.T) {
	dir := t.TempDir()
	write(t, filepath.Join(dir, "common.tdl"),
		"package common\n\nprimitive string\n\ntype Money {\n  amount: string\n}\n")

	src := "package p\n\n" +
		"import \"common.tdl\" as common\n\n" +
		"primitive string\n\n" +
		"type Order: Entity {\n" +
		"  id: string\n" +
		"  total: common.Money\n" +
		"}\n"
	main := filepath.Join(dir, "order.tdl")
	write(t, main, src)

	file, err := parser.Parse(main, strings.NewReader(src))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	var refs sema.References
	if _, diags := sema.Lower(file,
		sema.WithReferences(&refs),
		sema.WithLoader(sema.FSLoader{}),
	); len(diags) > 0 {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	ref, ok := refs.At(main, strings.Index(src, "common.Money"))
	if !ok {
		t.Fatal("no reference on common.Money")
	}
	if ref.Name != "common.Money" {
		t.Errorf("name = %q, want common.Money", ref.Name)
	}
	if ref.Target.Filename != "" {
		t.Errorf("target = %s, want none", ref.Target)
	}
}

// write puts a file where an import can find it.
func write(t *testing.T, path, src string) {
	t.Helper()

	if err := os.WriteFile(path, []byte(src), 0o600); err != nil {
		t.Fatalf("writing %s: %v", path, err)
	}
}

// TestSugarRecordsNoReference holds the rule that keeps a cursor honest: a
// name the author never wrote is not in the index.
//
// `[T]` lowers through the prelude's List, and recording that would put a
// reference to List across the brackets, so a cursor on `[` would jump
// into the prelude.
func TestSugarRecordsNoReference(t *testing.T) {
	const src = "package p\n\n" +
		"primitive string\n\n" +
		"type User: Entity {\n" +
		"  tags: [string]\n" +
		"}\n"

	refs := lowerWithRefs(t, src)

	off := strings.Index(src, "[string]")
	ref, ok := refs.At("refs.tdl", off)
	if ok && ref.Name != "string" {
		t.Errorf("the bracket resolved to %q; sugar should record nothing", ref.Name)
	}
}

// TestReferencesAreOffByDefault is why this is an option: nothing but an
// editor wants the table, and `tdl check` should not build one.
func TestReferencesAreOffByDefault(t *testing.T) {
	const src = "package p\n\nprimitive string\n\ntype Email: string\n"

	file, err := parser.Parse("refs.tdl", strings.NewReader(src))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if _, diags := sema.Lower(file); len(diags) > 0 {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	// Nothing to assert but that it lowers: the point is that Lower takes
	// no sink and allocates none.
}

// TestPreludeIsNotIndexed keeps the table to the file being edited. The
// prelude lowers through the same lowerer, and an editor should not search
// its names on every request.
func TestPreludeIsNotIndexed(t *testing.T) {
	const src = "package p\n\nprimitive string\n\ntype Email: string\n"

	for _, ref := range lowerWithRefs(t, src) {
		if ref.Pos.Filename != "refs.tdl" {
			t.Errorf("indexed a reference from %s", ref.Pos.Filename)
		}
	}
}
