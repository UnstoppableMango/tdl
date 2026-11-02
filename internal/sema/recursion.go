package sema

import (
	"slices"

	"github.com/unstoppablemango/tdl/ast"
	"github.com/unstoppablemango/tdl/ir"
)

// checkRecursion enforces the spec's three recursion rules.
//
//   - A struct conforming to Entity may be mutually recursive without
//     restriction.
//   - Any other struct, every enum, and every newtype must have a finite
//     value: `next: Node` gives `Node` none, while `next: Node?`, or a
//     field of an enum with a variant that ends the recursion, does. An
//     enum is never exempt, even conforming to Entity.
//   - Aliases may never be recursive, even through a collection.
//
// It runs after [lowerer.markEntities], which sets the entity kind.
func (l *lowerer) checkRecursion(file *ast.File) {
	finite := finiteDecls(file)
	for _, decl := range file.Decls {
		name := decl.Name()
		switch decl.(type) {
		case *ast.AliasDecl:
			// Expansion follows every edge, including through collections.
			l.findCycle(file, name, name, map[string]bool{}, nil)
		case *ast.StructDecl:
			if b, ok := l.file.lookup(name); ok && b.kind == bindDecl &&
				l.model.Decl(b.id).GetStructure().GetKind() == ir.StructKind_STRUCT_KIND_ENTITY {
				continue
			}
			if !finite[name] {
				l.findCycle(file, name, name, map[string]bool{}, finite)
			}
		case *ast.EnumDecl, *ast.NewtypeDecl:
			if !finite[name] {
				l.findCycle(file, name, name, map[string]bool{}, finite)
			}
		}
	}
}

// finiteDecls returns the declarations of a file that have a finite value,
// the least fixed point of:
//
//   - a struct is finite when every field's type is;
//   - an enum is finite when some variant's fields all are, so a fieldless
//     variant, or no variant at all, makes it finite;
//   - a newtype or an alias is finite when its base or target is.
//
// A type is finite when it is a collection or an optional, names no
// declaration of the file (a primitive, a prelude name, a type parameter,
// another package), or names a finite one with finite arguments.
//
// Entities are walked like any struct: one holding itself is not finite,
// and only reporting exempts it, so a value holding an entity that holds
// the value still has none.
func finiteDecls(file *ast.File) map[string]bool {
	decls := map[string]ast.Decl{}
	for _, d := range file.Decls {
		switch d.(type) {
		case *ast.AliasDecl, *ast.NewtypeDecl, *ast.StructDecl, *ast.EnumDecl:
			if _, dup := decls[d.Name()]; !dup {
				decls[d.Name()] = d
			}
		}
	}

	finite := map[string]bool{}
	for changed := true; changed; {
		changed = false
		for name, d := range decls {
			if !finite[name] && declFinite(d, decls, finite) {
				finite[name] = true
				changed = true
			}
		}
	}
	return finite
}

// declFinite reports whether a declaration is finite, given the
// declarations already known to be.
func declFinite(d ast.Decl, decls map[string]ast.Decl, finite map[string]bool) bool {
	params := typeParams(d)
	ok := func(t *ast.TypeRef) bool { return typeFinite(t, params, decls, finite) }

	switch d := d.(type) {
	case *ast.AliasDecl:
		return ok(d.Target)
	case *ast.NewtypeDecl:
		return ok(d.Base)
	case *ast.StructDecl:
		for _, m := range d.Members {
			if f, isField := m.(*ast.Field); isField && !ok(f.Type) {
				return false
			}
		}
		return true
	case *ast.EnumDecl:
		if len(d.Variants) == 0 {
			return true
		}
		for _, v := range d.Variants {
			if !slices.ContainsFunc(v.Fields, func(f *ast.Field) bool { return !ok(f.Type) }) {
				return true
			}
		}
	}
	return false
}

func typeFinite(t *ast.TypeRef, params map[string]bool, decls map[string]ast.Decl, finite map[string]bool) bool {
	if t == nil || wrapped(t) || t.Qualifier != "" {
		return true
	}
	if _, isDecl := decls[t.N]; isDecl && !params[t.N] && !finite[t.N] {
		return false
	}
	for _, a := range t.Args {
		if !typeFinite(a.Type, params, decls, finite) {
			return false
		}
	}
	return true
}

// typeParams returns the names a declaration's parameters bind, which
// shadow the file's declarations inside it.
func typeParams(d ast.Decl) map[string]bool {
	var ps []*ast.TypeParam
	switch d := d.(type) {
	case *ast.AliasDecl:
		ps = d.Params
	case *ast.NewtypeDecl:
		ps = d.Params
	case *ast.StructDecl:
		ps = d.Params
	case *ast.EnumDecl:
		ps = d.Params
	}
	out := map[string]bool{}
	for _, p := range ps {
		out[p.N] = true
	}
	return out
}

// findCycle walks from `at` looking for `start`. With `finite` nil, every
// edge counts, through collections too; otherwise only an edge outside a
// collection or an optional into a declaration with no finite value.
func (l *lowerer) findCycle(file *ast.File, start, at string, seen map[string]bool, finite map[string]bool) {
	if seen[at] {
		return
	}
	seen[at] = true

	for _, next := range l.edges(file, at, finite) {
		if next.name == start {
			if start == at {
				l.diags.add(next.at, "%s contains itself", start)
			} else {
				l.diags.add(next.at, "%s and %s contain each other", start, at)
			}
			return
		}
		l.findCycle(file, start, next.name, seen, finite)
	}
}

type edge struct {
	name string
	at   ast.Node
}

// edges returns the declarations a declaration reaches directly.
func (l *lowerer) edges(file *ast.File, name string, finite map[string]bool) []edge {
	decl := findDecl(file, name)
	if decl == nil {
		return nil
	}
	throughWrappers := finite == nil
	params := typeParams(decl)

	var out []edge
	add := func(t *ast.TypeRef) {
		if t == nil {
			return
		}
		if !throughWrappers && wrapped(t) {
			return // the wrapper gives it a finite value
		}
		for _, n := range reachedNames(t, throughWrappers) {
			if params[n] || (!throughWrappers && finite[n]) {
				continue
			}
			out = append(out, edge{name: n, at: t})
		}
	}

	switch d := decl.(type) {
	case *ast.AliasDecl:
		add(d.Target)
	case *ast.NewtypeDecl:
		add(d.Base)
	case *ast.StructDecl:
		for _, m := range d.Members {
			if f, ok := m.(*ast.Field); ok {
				add(f.Type)
			}
		}
	case *ast.EnumDecl:
		for _, v := range d.Variants {
			for _, f := range v.Fields {
				add(f.Type)
			}
		}
	}
	return out
}

// wrapped reports whether a type reference is a collection or an optional,
// as sugar or spelled out. `List<T>` names what `[T]` lowers to, so the two
// wrap alike.
func wrapped(t *ast.TypeRef) bool {
	return t.List != nil || t.Set != nil || t.MapKey != nil || t.Optional || t.Nullable || spelledWrapper(t)
}

// spelledWrapper reports whether a type reference names, with its
// arguments, a declaration sugar lowers to.
func spelledWrapper(t *ast.TypeRef) bool {
	if t.Qualifier != "" {
		return false
	}
	want := 1
	switch t.N {
	case preludeList, preludeSet, preludeOption, preludeNullable:
	case preludeMap:
		want = 2
	default:
		return false
	}
	return len(t.Args) == want
}

// reachedNames returns the declaration names a type reference mentions.
func reachedNames(t *ast.TypeRef, throughWrappers bool) []string {
	if t == nil {
		return nil
	}

	switch {
	case t.List != nil:
		if !throughWrappers {
			return nil
		}
		return reachedNames(t.List, throughWrappers)
	case t.Set != nil:
		if !throughWrappers {
			return nil
		}
		return reachedNames(t.Set, throughWrappers)
	case t.MapKey != nil:
		if !throughWrappers {
			return nil
		}
		return append(reachedNames(t.MapKey, throughWrappers), reachedNames(t.MapValue, throughWrappers)...)
	}

	if t.Qualifier != "" {
		return nil // another package, and imports are not resolved yet
	}
	if !throughWrappers && spelledWrapper(t) {
		return nil
	}

	names := []string{t.N}
	for _, a := range t.Args {
		if a.Type != nil {
			names = append(names, reachedNames(a.Type, throughWrappers)...)
		}
	}
	return names
}

func findDecl(file *ast.File, name string) ast.Decl {
	i := slices.IndexFunc(file.Decls, func(d ast.Decl) bool { return d.Name() == name })
	if i < 0 {
		return nil
	}
	return file.Decls[i]
}
