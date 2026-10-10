package ast

// Inspect traverses the tree rooted at n in source order. It calls f(n)
// and, when that returns true, inspects each child of n and then calls
// f(nil), as [go/ast.Inspect] does.
//
// Doc comments and ordinary comments are not nodes; they are on
// [DeclHead.Doc] and [File.Comments].
func Inspect(n Node, f func(Node) bool) {
	if !f(n) {
		return
	}
	for _, c := range Children(n) {
		Inspect(c, f)
	}
	f(nil)
}

// Children returns the direct children of n in source order, leaving out
// the ones that are absent.
func Children(n Node) []Node {
	var c children
	switch n := n.(type) {
	case *File:
		add(&c, n.Package)
		each(&c, n.Imports)
		each(&c, n.Decls)
	case *PrimitiveDecl:
		add(&c, n.Dep)
		add(&c, n.Kind)
	case *AliasDecl:
		add(&c, n.Dep)
		each(&c, n.Params)
		add(&c, n.Target)
	case *NewtypeDecl:
		add(&c, n.Dep)
		each(&c, n.Params)
		add(&c, n.Base)
		each(&c, n.Requires)
		each(&c, n.Constraints)
	case *StructDecl:
		add(&c, n.Dep)
		each(&c, n.Params)
		each(&c, n.Conforms)
		each(&c, n.Requires)
		each(&c, n.Members)
	case *EnumDecl:
		add(&c, n.Dep)
		each(&c, n.Params)
		each(&c, n.Conforms)
		each(&c, n.Requires)
		each(&c, n.Variants)
	case *TargetDecl:
		add(&c, n.Dep)
		each(&c, n.Entries)
	case *ClassDecl:
		add(&c, n.Dep)
		each(&c, n.Params)
		each(&c, n.FunDeps)
		each(&c, n.Conforms)
		each(&c, n.Requires)
		each(&c, n.Members)
	case *AssocTypeReq:
		add(&c, n.Dep)
		add(&c, n.Kind)
	case *InstanceDecl:
		add(&c, n.Dep)
		each(&c, n.Params)
		add(&c, n.Class)
		add(&c, n.For)
		each(&c, n.Requires)
		each(&c, n.Binds)
	case *UnitDecl:
		add(&c, n.Dep)
		add(&c, n.Expr)
	case *Field:
		add(&c, n.Dep)
		add(&c, n.Type)
		add(&c, n.Default)
		each(&c, n.Constraints)
	case *Variant:
		add(&c, n.Dep)
		each(&c, n.Fields)
	case *Include:
		add(&c, n.Type)
	case *TargetEntry:
		add(&c, n.Directive)
		each(&c, n.Entries)
	case *Directive:
		each(&c, n.Args)
	case *Constraint:
		each(&c, n.Args)
	case *Literal:
		each(&c, n.Items)
		add(&c, n.Lo)
		add(&c, n.Hi)
	case *AssocTypeBind:
		add(&c, n.Target)
	case *TypeParam:
		add(&c, n.Kind)
	case *Kind:
		add(&c, n.Paren)
		add(&c, n.Arrow)
	case *TypeRef:
		each(&c, n.Args)
		add(&c, n.List)
		add(&c, n.Set)
		add(&c, n.MapKey)
		add(&c, n.MapValue)
	case *ClassRef:
		each(&c, n.Args)
	case *TypeArg:
		add(&c, n.Type)
		add(&c, n.Unit)
	case *UnitExpr:
		each(&c, n.Terms)
	case *UnitTerm:
		add(&c, n.Paren)
	}
	return c
}

type children []Node

// add appends n unless it is absent.
func add[T any, P interface {
	*T
	Node
}](c *children, n P) {
	if n != nil {
		*c = append(*c, n)
	}
}

// each appends every node of ns, none of which is absent.
func each[T Node](c *children, ns []T) {
	for _, n := range ns {
		*c = append(*c, n)
	}
}
