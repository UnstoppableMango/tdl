package nix

import (
	"slices"

	"github.com/unstoppablemango/tdl/backend/internal/emit"
)

// setHelper is a list refusing an element twice. It checks the merged
// list, since two definitions, or two that differ only in an unset
// optional, may each hold one.
const setHelper = `  set =
    type:
    after (
      loc: l: if lib.allUnique l then l else throw "${lib.showOption loc} holds an element twice."
    ) (types.listOf type);
`

// wireStrings are the primitives JSON writes as a string of a known form.
var wireStrings = []string{"bytes", "decimal", "uuid", "date", "instant", "duration"}

// attrs is a map's type: an attribute set, whose names are strings, so a
// key is a string, a string's wire form, or a fieldless enum's tag.
func (g *generator) attrs(r *emit.Ref) (string, error) {
	val, err := g.typ(r.Elem)
	if err != nil {
		return "", err
	}
	k := r.Key
	switch {
	case k.Form == emit.Prim && k.Name == "string":
		return "types.attrsOf " + arg(val), nil
	case k.Form == emit.Prim && slices.Contains(wireStrings, k.Name),
		k.Form == emit.Named && k.Decl.GetEnumeration() != nil && !emit.Fielded(k.Decl.GetEnumeration()):
		key, err := g.typ(k)
		if err != nil {
			return "", err
		}
		g.used["keyed"] = true
		return "keyed " + arg(key) + " " + arg(val), nil
	}
	return "", emit.Lost(emit.LossCollection, r.Pos, "an attribute name is a string, and this map's key is not one")
}
