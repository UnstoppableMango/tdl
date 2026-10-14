package csharp

import (
	"github.com/unstoppablemango/tdl/backend/internal/dotnet"
	"github.com/unstoppablemango/tdl/backend/internal/emit"
	"github.com/unstoppablemango/tdl/ir"
)

// keywordTypes is the C# keyword spelling each base class library type has.
var keywordTypes = map[string]string{
	"String":  "string",
	"Int64":   "long",
	"Int32":   "int",
	"UInt32":  "uint",
	"UInt64":  "ulong",
	"Single":  "float",
	"Double":  "double",
	"Boolean": "bool",
	"Byte":    "byte",
	"Decimal": "decimal",
}

const (
	immutable  = "global::System.Collections.Immutable."
	generic    = "global::System.Collections.Generic."
	listType   = immutable + "IImmutableList"
	setType    = immutable + "IImmutableSet"
	mapType    = immutable + "IImmutableDictionary"
	comparerOf = generic + "EqualityComparer"
)

// csType is a rendered type.
type csType struct {
	text string

	// cmp is the comparer that compares it by content, or "" when
	// EqualityComparer<T>.Default already does.
	cmp string
}

// defaultComparer is the comparer a record compares a property of type typ
// by.
func defaultComparer(typ string) string {
	return comparerOf + "<" + typ + ">.Default"
}

// fieldType renders a field's type and whether the property is required:
// `T?` may be left out, `T | null` must be written and may be null.
func (g *generator) fieldType(f *ir.Field, shadow map[string]bool) (csType, bool, error) {
	ref, err := g.Resolve(f.GetType())
	if err != nil {
		return csType{}, false, err
	}
	required := true
	switch ref.Form {
	case emit.Option:
		required = false
	case emit.Nullable:
		if ref.Elem.Form == emit.Option {
			required = false
		}
	}
	t, err := g.render(ref, shadow)
	return t, required, err
}

// typ renders a type reference.
func (g *generator) typ(id *ir.ID, shadow map[string]bool) (csType, error) {
	ref, err := g.Resolve(id)
	if err != nil {
		return csType{}, err
	}
	return g.render(ref, shadow)
}

// render renders a resolved type. A type whose name a nested variant
// shadows, named in shadow, is written qualified.
func (g *generator) render(r *emit.Ref, shadow map[string]bool) (csType, error) {
	switch r.Form {
	case emit.Prim:
		if t := g.Model.Type(r.ID); len(t.GetArgs()) > 0 {
			g.Lossy(emit.LossUnit, r.Pos, "C# has no units of measure, so %s is written as its base type", r.ID.GetName())
		}
		t, ok := dotnet.Primitive(r.Name)
		if !ok {
			return csType{}, emit.Lost(emit.LossPrimitive, r.Pos, "primitive %s has no C# type", r.Name)
		}
		text := spell(t)
		if t.Array {
			return csType{text: text + "[]", cmp: g.equality() + ".Bytes"}, nil
		}
		return csType{text: text}, nil

	case emit.Option, emit.Nullable:
		elem := r.Elem
		if elem.Form == emit.Option || elem.Form == emit.Nullable {
			// C# has one null, so an absent value and a null one are the
			// same.
			g.Lossy(emit.LossOptional, r.Pos, "C# cannot tell an absent value from a null one, so both are null")
			for elem.Form == emit.Option || elem.Form == emit.Nullable {
				elem = elem.Elem
			}
		}
		t, err := g.render(elem, shadow)
		if err != nil {
			return csType{}, err
		}
		t.text += "?"
		return t, nil

	case emit.List:
		elem, err := g.render(r.Elem, shadow)
		if err != nil {
			return csType{}, err
		}
		return csType{
			text: listType + "<" + elem.text + ">",
			cmp:  g.equality() + ".List<" + elem.text + ">(" + elem.comparer() + ")",
		}, nil

	case emit.Set:
		if err := hashed(r.Elem, "a set's element"); err != nil {
			return csType{}, err
		}
		elem, err := g.render(r.Elem, shadow)
		if err != nil {
			return csType{}, err
		}
		return csType{
			text: setType + "<" + elem.text + ">",
			cmp:  g.equality() + ".Set<" + elem.text + ">()",
		}, nil

	case emit.Map:
		if err := hashed(r.Key, "a map's key"); err != nil {
			return csType{}, err
		}
		key, err := g.render(r.Key, shadow)
		if err != nil {
			return csType{}, err
		}
		val, err := g.render(r.Elem, shadow)
		if err != nil {
			return csType{}, err
		}
		return csType{
			text: mapType + "<" + key.text + ", " + val.text + ">",
			cmp:  g.equality() + ".Map<" + key.text + ", " + val.text + ">(" + val.comparer() + ")",
		}, nil

	case emit.Named:
		name, ok := g.names[r.Decl]
		if !ok {
			// A declaration that does not claim a name is skipped, and the
			// cascade skips this one.
			return csType{}, emit.Unsupported(r.Pos, "%s is not generated", r.Decl.GetMeta().GetName())
		}
		if shadow[name] {
			name = "global::" + g.ns + "." + name
		}
		return csType{text: name}, nil
	}
	return csType{}, emit.Unsupported(r.Pos, "type %s has no C# type", r.ID.GetName())
}

// comparer is the expression comparing a value of the type by content.
func (t csType) comparer() string {
	if t.cmp != "" {
		return t.cmp
	}
	return defaultComparer(t.text)
}

// hashed refuses, as a set's element or a map's key, a type .NET hashes by
// reference or one that may be null.
func hashed(r *emit.Ref, what string) error {
	switch {
	case r.Form == emit.Option || r.Form == emit.Nullable:
		return emit.Unsupported(r.Pos, "%s cannot be optional in C#", what)
	case r.Form == emit.List || r.Form == emit.Set || r.Form == emit.Map:
		return emit.Unsupported(r.Pos, "%s cannot be a collection in C#, which hashes it by reference", what)
	case r.Form == emit.Prim && r.Name == "bytes":
		return emit.Unsupported(r.Pos, "%s cannot be bytes in C#, which hashes it by reference", what)
	}
	return nil
}

// spell is a base class library type as C# writes it.
func spell(t dotnet.Type) string {
	if t.Namespace == "System" {
		if k, ok := keywordTypes[t.Name]; ok {
			return k
		}
	}
	return "global::" + t.Namespace + "." + t.Name
}

// equality is the namespace's equality class, qualified, since a nested
// variant may shadow it.
func (g *generator) equality() string {
	return "global::" + g.ns + "." + equalityType
}
