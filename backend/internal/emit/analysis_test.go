package emit_test

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/unstoppablemango/tdl/backend/internal/emit"
	"github.com/unstoppablemango/tdl/internal/sema"
	"github.com/unstoppablemango/tdl/ir"
	"github.com/unstoppablemango/tdl/parser"
)

// TestAnalysisMatchesTheWalk holds the analysis-backed Resolve and
// References to what walking the model directly gives, over the
// conformance corpus and the smoke model, so moving the walk into the host
// changes no backend's output.
func TestAnalysisMatchesTheWalk(t *testing.T) {
	paths, err := filepath.Glob("../../../testdata/conformance/*/source.tdl")
	if err != nil {
		t.Fatal(err)
	}
	paths = append(paths, "../../../testdata/gen/smoke/source.tdl")

	for _, path := range paths {
		if _, err := os.Stat(filepath.Join(filepath.Dir(path), "pending")); err == nil {
			continue
		}
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		file, err := parser.Parse(path, strings.NewReader(string(data)))
		if err != nil {
			t.Fatal(err)
		}
		model, diags := sema.Lower(file, sema.WithLoader(sema.FSLoader{}))
		if len(diags) > 0 {
			t.Fatalf("%s: %v", path, diags)
		}

		for _, externs := range []bool{false, true} {
			s := &emit.Session{Model: model, Target: "go", Lang: "Go", Externs: externs}
			for i := range model.GetTypes() {
				id := &ir.ID{Index: int32(i), Name: fmt.Sprint("t", i)}
				got, gotErr := s.Resolve(id)
				want, wantErr := walkResolve(model, "Go", externs, id)
				if g, w := describe(got, gotErr), describe(want, wantErr); g != w {
					t.Errorf("%s: type %d (externs %v):\n got %s\nwant %s", path, i, externs, g, w)
				}
			}
		}

		s := &emit.Session{Model: model, Target: "go", Lang: "Go"}
		var all []*ir.ID
		for _, d := range model.GetDecls() {
			if g, w := names(s.References(d)), names(walkReferences(model, bodyTypes(d)...)); g != w {
				t.Errorf("%s: References(%s) = %s, want %s", path, d.GetMeta().GetName(), g, w)
			}
			all = append(all, bodyTypes(d)...)
		}
		if g, w := names(s.TypeReferences(all...)), names(walkReferences(model, all...)); g != w {
			t.Errorf("%s: TypeReferences = %s, want %s", path, g, w)
		}
	}
}

// TestAnalysisFromTheRequest checks a session reads the analysis a request
// carries rather than deriving its own.
func TestAnalysisFromTheRequest(t *testing.T) {
	file, err := parser.Parse("x.tdl", strings.NewReader("package x\n\ntype A { b: string }\n"))
	if err != nil {
		t.Fatal(err)
	}
	model, diags := sema.Lower(file)
	if len(diags) > 0 {
		t.Fatal(diags)
	}
	s := &emit.Session{Model: model, Target: "go", Lang: "Go"}
	if len(s.Own()) != 1 {
		t.Fatalf("Own = %d declarations, want 1", len(s.Own()))
	}

	// A request whose analysis claims nothing is the model's own is
	// believed, which shows the session read it.
	for _, d := range s.Analysis.GetDecls() {
		d.Own = false
	}
	if own := s.Own(); len(own) != 0 {
		t.Errorf("Own = %d declarations, want the request's 0", len(own))
	}
}

func describe(r *emit.Ref, err error) string {
	if err != nil {
		var u *emit.UnsupportedError
		if errors.As(err, &u) {
			return fmt.Sprintf("error %q at %v", u.What, u.Position)
		}
		return "error " + err.Error()
	}
	if r == nil {
		return "nil"
	}
	return fmt.Sprintf("{form %d name %q decl %q extern %q id %d pos %v elem %s key %s}",
		r.Form, r.Name, r.Decl.GetMeta().GetName(), r.Extern.GetName(), r.ID.GetIndex(), r.Pos,
		describe(r.Elem, nil), describe(r.Key, nil))
}

func names(ds []*ir.Decl) string {
	var out []string
	for _, d := range ds {
		out = append(out, d.GetMeta().GetName())
	}
	return strings.Join(out, ",")
}

func bodyTypes(d *ir.Decl) []*ir.ID {
	var ids []*ir.ID
	switch {
	case d.GetStructure() != nil:
		for _, f := range d.Fields() {
			ids = append(ids, f.GetType())
		}
	case d.GetEnumeration() != nil:
		for _, v := range d.GetEnumeration().GetVariants() {
			for _, f := range v.GetFields() {
				ids = append(ids, f.GetType())
			}
		}
	case d.GetNewtype() != nil:
		ids = append(ids, d.GetNewtype().GetBase())
	case d.GetAlias() != nil:
		ids = append(ids, d.GetAlias().GetTarget())
	}
	return ids
}

// walkResolve is emit.Resolve as it was before the analysis, walking the
// model directly.
func walkResolve(m *ir.Model, lang string, externs bool, id *ir.ID) (*emit.Ref, error) {
	t := m.Type(id)
	if t == nil {
		return nil, emit.Unsupported(nil, "type %s did not resolve", id.GetName())
	}
	pos := t.GetPosition()

	switch {
	case t.GetParam() != nil:
		return nil, emit.Unsupported(pos, "type parameter %s: generics are not generated yet", t.GetParam().GetName())
	case t.GetUnit() != nil:
		return nil, emit.Unsupported(pos, "a unit-typed field has no %s type yet", lang)
	case t.GetExtern() != nil:
		ext := t.GetExtern()
		if i := int(ext.GetIndex()); externs && i >= 0 && i < len(m.GetExterns()) {
			return &emit.Ref{Form: emit.Extern, Name: ext.GetName(), Extern: m.GetExterns()[i], ID: id, Pos: pos}, nil
		}
		return nil, emit.Unsupported(pos, "%s is declared in another package, and foreign types are not generated yet", ext.GetName())
	}

	decl := m.Decl(t.GetCtor())
	if decl == nil {
		return nil, emit.Unsupported(pos, "type %s did not resolve", t.GetCtor().GetName())
	}
	name := decl.GetMeta().GetName()

	if a := decl.GetAlias(); a != nil {
		return walkResolve(m, lang, externs, a.GetTarget())
	}

	ref := &emit.Ref{ID: id, Pos: pos}
	args := t.GetArgs()
	arg := func(i int) (*emit.Ref, error) {
		if i >= len(args) {
			return nil, emit.Unsupported(pos, "%s is missing a type argument", name)
		}
		return walkResolve(m, lang, externs, args[i])
	}

	var err error
	if decl.GetPrimitive() != nil {
		switch name {
		case "List", "Set":
			ref.Form = emit.List
			if name == "Set" {
				ref.Form = emit.Set
			}
			ref.Elem, err = arg(0)
		case "Map":
			ref.Form = emit.Map
			if ref.Key, err = arg(0); err == nil {
				ref.Elem, err = arg(1)
			}
		default:
			ref.Form, ref.Name = emit.Prim, name
		}
		if err != nil {
			return nil, err
		}
		return ref, nil
	}

	if decl.GetEnumeration() != nil && (name == "Option" || name == "Nullable") && len(args) == 1 {
		ref.Form = emit.Option
		if name == "Nullable" {
			ref.Form = emit.Nullable
		}
		if ref.Elem, err = arg(0); err != nil {
			return nil, err
		}
		return ref, nil
	}

	if decl.GetClass() != nil {
		return nil, emit.Unsupported(pos, "%s is a class, and a class is not a %s type", name, lang)
	}
	if decl.GetUnit() != nil {
		return nil, emit.Unsupported(pos, "%s is a unit, and units are not generated yet", name)
	}
	if len(args) > 0 {
		return nil, emit.Unsupported(pos, "%s is applied to type arguments, and generics are not generated yet", name)
	}

	ref.Form, ref.Decl = emit.Named, decl
	return ref, nil
}

// walkReferences is emit.TypeReferences as it was before the analysis.
func walkReferences(m *ir.Model, ids ...*ir.ID) []*ir.Decl {
	var out []*ir.Decl
	seen := map[*ir.Decl]bool{}
	var walk func(id *ir.ID)
	walk = func(id *ir.ID) {
		t := m.Type(id)
		if t == nil {
			return
		}
		for _, arg := range t.GetArgs() {
			walk(arg)
		}
		decl := m.Decl(t.GetCtor())
		if decl == nil || seen[decl] {
			return
		}
		seen[decl] = true
		if a := decl.GetAlias(); a != nil {
			walk(a.GetTarget())
			return
		}
		out = append(out, decl)
	}
	for _, id := range ids {
		walk(id)
	}
	return out
}
