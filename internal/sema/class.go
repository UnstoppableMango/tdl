package sema

import (
	"github.com/unstoppablemango/tdl/ast"
	"github.com/unstoppablemango/tdl/ir"
)

// classNode lowers a class declaration.
func (l *lowerer) classNode(d *ast.ClassDecl) *ir.Class {
	c := &ir.Class{
		Params:          l.params(d.Params),
		RequiresClasses: l.classRefs(d.Conforms),
		Constraints:     l.classRefs(d.Requires),
	}

	for _, dep := range d.FunDeps {
		c.FunDeps = append(c.FunDeps, &ir.FunDep{
			From:     dep.From,
			To:       dep.To,
			Position: position(dep.P),
		})
	}

	var fields []*ast.Field
	for _, m := range d.Members {
		switch member := m.(type) {
		case *ast.KeyRequirement:
			c.RequiresKey = true
		case *ast.AssocTypeReq:
			c.AssocTypes = append(c.AssocTypes, &ir.AssocType{
				Meta: metaOf(member.N, member.Doc, member.P, nil, len(c.AssocTypes)),
				Kind: kind(member.Kind),
			})
		case *ast.Field:
			fields = append(fields, member)
		}
	}
	c.Fields = l.variantFields(fields)
	return c
}

// instance lowers a standalone instance declaration.
//
// `instance C for T` is sugar for `instance C<T>`, so the `for` form
// becomes an argument here and there is one form from this point on.
func (l *lowerer) instance(d *ast.InstanceDecl, order int) *ir.Instance {
	inst := &ir.Instance{Meta: metaOf(d.Class.N, d.Doc, d.P, d.Dep, order)}

	s := l.paramScope(nil, d.Params)
	l.inScope(s, func() {
		inst.Params = l.params(d.Params)
		inst.Class = l.classRef(d.Class)
		if d.For != nil {
			inst.Class.Args = append(inst.Class.Args, l.typeRef(d.For))
		}
		inst.Requires = l.classRefs(d.Requires)

		for _, b := range d.Binds {
			inst.Binds = append(inst.Binds, &ir.AssocBind{
				Name:     b.N,
				Type:     l.typeRef(b.Target),
				Position: position(b.P),
			})
		}
	})
	return inst
}

func (l *lowerer) classRefs(refs []*ast.ClassRef) []*ir.ClassRef {
	var out []*ir.ClassRef
	for _, r := range refs {
		out = append(out, l.classRef(r))
	}
	return out
}

// classRef resolves a class name. A class is a declaration like any other,
// so it resolves through the same scope; a qualified one is an extern.
func (l *lowerer) classRef(r *ast.ClassRef) *ir.ClassRef {
	out := &ir.ClassRef{Position: position(r.P)}
	for _, a := range r.Args {
		if a.Unit != nil {
			l.diags.add(a.P, "unit arguments are not lowered yet")
			out.Args = append(out.Args, &ir.ID{Index: ir.Unresolved})
			continue
		}
		out.Args = append(out.Args, l.typeRef(a.Type))
	}

	if r.Qualifier != "" {
		pkg, ok := l.aliases[r.Qualifier]
		if !ok {
			l.diags.add(r.P, "undefined import alias: %s", r.Qualifier)
			return out
		}
		out.Extern = l.extern(pkg, r.N, r.P)
		return out
	}

	b, ok := l.scope.lookup(r.N)
	switch {
	case ok && b.kind == bindExtern:
		out.Extern = b.id
	case ok && b.kind == bindDecl:
		out.Class = b.id
	default:
		l.diags.add(r.P, "undefined class: %s", r.N)
		out.Class = &ir.ID{Index: ir.Unresolved, Name: r.N}
	}
	return out
}
