package sema

import (
	"testing"

	"github.com/unstoppablemango/tdl/prelude"
)

// The prelude declares the fixed-width numerics, so a field may name each
// one without the file declaring it.
func TestFixedWidthNumericsResolveToPrelude(t *testing.T) {
	model := lower(t, `
type Reading {
  a: int32
  b: uint32
  c: int64
  d: uint64
  e: float32
  f: float64
}
`)

	reading, _, ok := model.FindDecl("Reading")
	if !ok {
		t.Fatal("Reading is missing from the table")
	}
	want := []string{"int32", "uint32", "int64", "uint64", "float32", "float64"}
	fields := reading.Fields()
	if len(fields) != len(want) {
		t.Fatalf("Reading has %d fields, want %d", len(fields), len(want))
	}

	for i, name := range want {
		t.Run(name, func(t *testing.T) {
			ctor := model.Type(fields[i].GetType()).GetCtor()
			if !ctor.Resolved() || ctor.GetName() != name {
				t.Fatalf("field %d did not resolve to %s: %+v", i, name, ctor)
			}
			decl := model.Decl(ctor)
			if decl.GetPrimitive() == nil {
				t.Errorf("%s is not a primitive", name)
			}
			if got := decl.GetMeta().GetPosition().GetFilename(); got != prelude.Name {
				t.Errorf("%s came from %q, want %q", name, got, prelude.Name)
			}
		})
	}
}
