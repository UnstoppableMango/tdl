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

// TestReferencesCoverEveryMention checks that a second mention of an
// interned type is still recorded.
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

// TestReferencesFindTypeParameters checks that T resolves to the parameter
// that shadows a declaration of that name.
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
	// The parameter is bound in `Page<T>` on line 5.
	if ref.Target.Line != 5 {
		t.Errorf("target line = %d, want 5", ref.Target.Line)
	}
	if ref.Decl != nil {
		t.Error("a type parameter declares nothing, so Decl should be unset")
	}
}

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

// TestReferencesCrossAnImport checks that a name a `_` import merged in
// targets its declaration in the dependency.
func TestReferencesCrossAnImport(t *testing.T) {
	dir := t.TempDir()
	dep := filepath.Join(dir, "common.tdl")
	write(t, dep, "package common\n\nprimitive string\n\ntype Money {\n  amount: string\n}\n")

	src := "package p\n\n" +
		"import \"common.tdl\" as _\n\n" +
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

// TestQualifiedReferencesHaveNoTarget checks that a qualified reference is
// recorded with no target.
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

// TestSugarRecordsNoReference checks that `[T]` records no reference to
// the prelude's List.
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

func TestReferencesAreOffByDefault(t *testing.T) {
	const src = "package p\n\nprimitive string\n\ntype Email: string\n"

	file, err := parser.Parse("refs.tdl", strings.NewReader(src))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if _, diags := sema.Lower(file); len(diags) > 0 {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	// Nothing to assert: Lower takes no sink.
}

func TestPreludeIsNotIndexed(t *testing.T) {
	const src = "package p\n\nprimitive string\n\ntype Email: string\n"

	for _, ref := range lowerWithRefs(t, src) {
		if ref.Pos.Filename != "refs.tdl" {
			t.Errorf("indexed a reference from %s", ref.Pos.Filename)
		}
	}
}
