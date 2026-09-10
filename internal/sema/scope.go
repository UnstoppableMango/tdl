package sema

import (
	"github.com/unstoppablemango/tdl/ast"
	"github.com/unstoppablemango/tdl/ir"
)

// bindingKind distinguishes what a name in scope refers to.
type bindingKind int

const (
	bindDecl   bindingKind = iota // a file-level declaration
	bindParam                     // a type parameter of the enclosing declaration
	bindExtern                    // a declaration merged in by a `_` import
)

type binding struct {
	kind  bindingKind
	id    *ir.ID       // set for bindDecl
	owner *ir.ID       // set for bindParam: the declaration that declares it
	index int32        // set for bindParam: position in the parameter list
	pos   ast.Position // where the name was bound
}

// scope is a chain of name bindings. The file's declarations sit at the
// root; a declaration's type parameters sit in a child scope, so a
// parameter named the same as a declaration shadows it inside that
// declaration and nowhere else.
type scope struct {
	parent *scope
	names  map[string]binding
}

func newScope(parent *scope) *scope {
	return &scope{parent: parent, names: map[string]binding{}}
}

// bind adds a name, reporting whether it was already bound in this scope.
// Shadowing an outer scope is allowed; rebinding within one is not.
func (s *scope) bind(name string, b binding) (binding, bool) {
	if existing, dup := s.names[name]; dup {
		return existing, false
	}
	s.names[name] = b
	return b, true
}

func (s *scope) lookup(name string) (binding, bool) {
	for cur := s; cur != nil; cur = cur.parent {
		if b, ok := cur.names[name]; ok {
			return b, true
		}
	}
	return binding{}, false
}

// paramScope binds a declaration's type parameters over the file scope.
func (l *lowerer) paramScope(owner *ir.ID, params []*ast.TypeParam) *scope {
	if len(params) == 0 {
		return l.file
	}

	s := newScope(l.file)
	for i, p := range params {
		if prev, ok := s.bind(p.N, binding{
			kind:  bindParam,
			owner: owner,
			index: int32(i),
			pos:   p.P,
		}); !ok {
			l.diags.add(p.P, "type parameter %s is declared twice, first at %s", p.N, prev.pos)
		}
	}
	return s
}
