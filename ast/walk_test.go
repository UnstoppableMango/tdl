package ast

import (
	"reflect"
	"testing"
)

// TestChildrenCoverEveryField fills every node-valued field of each node
// type and checks Children returns all of them, so a field added to a
// node cannot be missed by Inspect.
func TestChildrenCoverEveryField(t *testing.T) {
	nodeType := reflect.TypeFor[Node]()
	for _, n := range []Node{
		&File{}, &PackageDecl{}, &ImportDecl{}, &PrimitiveDecl{}, &AliasDecl{},
		&NewtypeDecl{}, &StructDecl{}, &EnumDecl{}, &TargetDecl{}, &ClassDecl{},
		&AssocTypeReq{}, &InstanceDecl{}, &UnitDecl{}, &Field{}, &Variant{},
		&Include{}, &TargetEntry{}, &Directive{}, &Constraint{}, &Literal{},
		&FunDep{}, &AssocTypeBind{}, &TypeParam{}, &Kind{}, &TypeRef{},
		&ClassRef{}, &TypeArg{}, &UnitExpr{}, &UnitTerm{}, &Deprecation{},
	} {
		v := reflect.ValueOf(n).Elem()
		want := fill(v, nodeType)
		if got := len(Children(n)); got != want {
			t.Errorf("%T: Children returned %d nodes, want %d", n, got, want)
		}
	}
}

// fill sets every node-valued field of the struct v, descending into
// embedded structs, and returns how many nodes it set.
func fill(v reflect.Value, nodeType reflect.Type) int {
	set := 0
	for i := range v.NumField() {
		f, sf := v.Field(i), v.Type().Field(i)
		switch {
		case sf.Anonymous && f.Kind() == reflect.Struct:
			set += fill(f, nodeType)
		case f.Kind() == reflect.Pointer && f.Type().Implements(nodeType):
			f.Set(reflect.New(f.Type().Elem()))
			set++
		case f.Kind() == reflect.Slice && f.Type().Elem().Implements(nodeType):
			elem := f.Type().Elem()
			if elem.Kind() == reflect.Interface {
				elem = reflect.TypeFor[*Field]()
			}
			f.Set(reflect.Append(f, reflect.New(elem.Elem())))
			set++
		}
	}
	return set
}

func TestInspectPrunes(t *testing.T) {
	ref := &TypeRef{N: "Map", Args: []*TypeArg{{Type: &TypeRef{N: "K"}}, {Type: &TypeRef{N: "V"}}}}
	file := &File{Decls: []Decl{&AliasDecl{DeclHead: DeclHead{N: "A"}, Target: ref}}}

	var visited []string
	Inspect(file, func(n Node) bool {
		if r, ok := n.(*TypeRef); ok {
			visited = append(visited, r.N)
		}
		_, arg := n.(*TypeArg)
		return !arg
	})
	if want := []string{"Map"}; !reflect.DeepEqual(visited, want) {
		t.Errorf("visited %v, want %v", visited, want)
	}
}
