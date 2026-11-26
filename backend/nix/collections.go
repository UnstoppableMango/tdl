package nix

import (
	"strings"

	"github.com/unstoppablemango/tdl/backend/internal/emit"
)

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
	case k.Form == emit.Prim && strings.HasPrefix(prims[k.Name], "types.strMatching"),
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
