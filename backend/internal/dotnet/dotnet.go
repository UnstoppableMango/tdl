// Package dotnet holds what the csharp and fsharp backends share because
// both target .NET: the base class library type each primitive maps to and
// the namespace a package derives. It holds decisions, not syntax; each
// backend spells a type its own way. See docs/design/csharp-backend.md.
package dotnet

import (
	"strings"
	"unicode"
)

// Type is a base class library type.
type Type struct {
	// Namespace and Name name the type, as System and Int64.
	Namespace, Name string

	// Array is whether the primitive is an array of the type, as bytes is
	// of System.Byte.
	Array bool
}

// primitives maps a TDL primitive to the type holding it.
var primitives = map[string]Type{
	"string":   {"System", "String", false},
	"int":      {"System", "Int64", false},
	"int32":    {"System", "Int32", false},
	"uint32":   {"System", "UInt32", false},
	"int64":    {"System", "Int64", false},
	"uint64":   {"System", "UInt64", false},
	"float32":  {"System", "Single", false},
	"float64":  {"System", "Double", false},
	"bool":     {"System", "Boolean", false},
	"bytes":    {"System", "Byte", true},
	"decimal":  {"System", "Decimal", false},
	"uuid":     {"System", "Guid", false},
	"instant":  {"System", "DateTimeOffset", false},
	"date":     {"System", "DateOnly", false},
	"duration": {"System", "TimeSpan", false},
}

// Primitive returns the type a TDL primitive maps to, and false for a
// primitive .NET has no type for.
func Primitive(name string) (Type, bool) {
	t, ok := primitives[name]
	return t, ok
}

// Namespace is the namespace a package derives: its segments, each with its
// first letter upper-cased, so shop.billing is Shop.Billing.
func Namespace(pkg string) string {
	segs := strings.Split(pkg, ".")
	for i, s := range segs {
		r := []rune(s)
		if len(r) > 0 {
			r[0] = unicode.ToUpper(r[0])
		}
		segs[i] = string(r)
	}
	return strings.Join(segs, ".")
}
