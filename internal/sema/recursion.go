package sema

import (
	"github.com/unstoppablemango/tdl/ast"
)

// checkRecursion enforces the spec's three recursion rules.
//
//   - Entities may be mutually recursive without restriction. A cycle
//     between entities is a graph of references, which every backend can
//     represent.
//   - A value may reach itself only through a collection or an optional,
//     never as a bare field. `value Node { next: Node }` has no finite
//     representation; `next: Node?` and `children: [Node]` do.
//   - Aliases may never be recursive, since they are expanded rather than
//     referenced. An alias cycle does not terminate under expansion, so
//     even a collection does not save it.
//
// The two rules differ in which edges they follow, so the walk is
// parameterized by that rather than duplicated.
func (l *lowerer) checkRecursion(file *ast.File) {
	for _, decl := range file.Decls {
		switch d := decl.(type) {
		case *ast.AliasDecl:
			// Expansion follows every edge, including through collections.
			l.findCycle(file, decl.Name(), d.Name(), map[string]bool{}, true)
		case *ast.StructDecl:
			if d.Keyword == "entity" {
				continue
			}
			l.findCycle(file, decl.Name(), d.Name(), map[string]bool{}, false)
		case *ast.EnumDecl:
			l.findCycle(file, decl.Name(), d.Name(), map[string]bool{}, false)
		}
	}
}

// findCycle walks from `at` looking for `start`. `throughWrappers` says
// whether an edge into a collection or optional counts.
func (l *lowerer) findCycle(file *ast.File, start, at string, seen map[string]bool, throughWrappers bool) {
	if seen[at] {
		return
	}
	seen[at] = true

	for _, next := range l.edges(file, at, throughWrappers) {
		if next.name == start {
			if start == at {
				l.diags.add(next.pos, "%s contains itself", start)
			} else {
				l.diags.add(next.pos, "%s and %s contain each other", start, at)
			}
			return
		}
		l.findCycle(file, start, next.name, seen, throughWrappers)
	}
}

type edge struct {
	name string
	pos  ast.Position
}

// edges returns the declarations a declaration reaches directly.
func (l *lowerer) edges(file *ast.File, name string, throughWrappers bool) []edge {
	decl := findDecl(file, name)
	if decl == nil {
		return nil
	}

	var out []edge
	add := func(t *ast.TypeRef) {
		if t == nil {
			return
		}
		if !throughWrappers && wrapped(t) {
			return // the wrapper gives it a finite representation
		}
		for _, n := range reachedNames(t, throughWrappers) {
			out = append(out, edge{name: n, pos: t.P})
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
// which is what gives a recursive value a finite representation.
func wrapped(t *ast.TypeRef) bool {
	return t.List != nil || t.Set != nil || t.MapKey != nil || t.Optional || t.Nullable
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

	names := []string{t.N}
	for _, a := range t.Args {
		if a.Type != nil {
			names = append(names, reachedNames(a.Type, throughWrappers)...)
		}
	}
	return names
}

func findDecl(file *ast.File, name string) ast.Decl {
	for _, d := range file.Decls {
		if d.Name() == name {
			return d
		}
	}
	return nil
}
