package sema

import (
	"sort"

	"github.com/unstoppablemango/tdl/ast"
	"github.com/unstoppablemango/tdl/ir"
)

// Reference is one name, as written, and what it resolved to. The type
// table interns a second mention away, so this is what "go to definition"
// reads.
type Reference struct {
	Pos  ast.Position // where the name was written
	Len  int          // its length in bytes
	Name string       // the name as written

	// Target is where the name's referent was declared, as bound in scope.
	// It is zero for an unresolved or qualified name. A name a `_` import
	// merged in points into the dependency that declares it.
	Target ast.Position

	// Decl is the declaration the name refers to, unset for a type
	// parameter and for an unresolved name.
	Decl *ir.ID
}

// References is every name one lowering resolved, in source order.
type References []Reference

// At returns the reference covering a byte offset in a file, and whether
// there is one. It is a binary search over the sorted table.
func (rs References) At(filename string, offset int) (Reference, bool) {
	i := sort.Search(len(rs), func(i int) bool {
		if rs[i].Pos.Filename != filename {
			return rs[i].Pos.Filename > filename
		}
		return rs[i].Pos.Offset+rs[i].Len > offset
	})
	if i == len(rs) {
		return Reference{}, false
	}

	r := rs[i]
	if r.Pos.Filename != filename || offset < r.Pos.Offset {
		return Reference{}, false
	}
	return r, true
}

// WithReferences records every name lowering resolves into out.
func WithReferences(out *References) Option {
	return func(c *config) { c.refs = out }
}

// record adds a reference, if this lowering is recording them. It is called
// where a name is written, so `[T]` records no `List`.
func (l *lowerer) record(pos ast.Position, name string, b binding, ok bool) {
	if l.refs == nil {
		return
	}

	ref := Reference{Pos: pos, Len: len(name), Name: name}
	if ok {
		ref.Target = b.pos
		if b.kind == bindDecl {
			ref.Decl = b.id
		}
	}
	*l.refs = append(*l.refs, ref)
}

// recordLookup resolves a name in the current scope and records it.
func (l *lowerer) recordLookup(pos ast.Position, name string) {
	if l.refs == nil {
		return
	}
	b, ok := l.scope.lookup(name)
	l.record(pos, name, b, ok)
}

// sort puts the table in source order for At. Later passes record out of
// order.
func (rs References) sort() {
	sort.SliceStable(rs, func(i, j int) bool {
		if rs[i].Pos.Filename != rs[j].Pos.Filename {
			return rs[i].Pos.Filename < rs[j].Pos.Filename
		}
		return rs[i].Pos.Offset < rs[j].Pos.Offset
	})
}
