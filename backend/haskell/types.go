package haskell

import (
	"regexp"
	"slices"
	"strings"

	"github.com/unstoppablemango/tdl/backend/internal/emit"
	"github.com/unstoppablemango/tdl/ir"
)

// imported is a type or class a module imports: the module it comes from
// and its name there.
type imported struct{ module, name string }

// scalars maps a TDL primitive to the Haskell type holding it.
var scalars = map[string]imported{
	"string":   {"Data.Text", "Text"},
	"int":      {"Data.Int", "Int64"},
	"int32":    {"Data.Int", "Int32"},
	"uint32":   {"Data.Word", "Word32"},
	"int64":    {"Data.Int", "Int64"},
	"uint64":   {"Data.Word", "Word64"},
	"float32":  {"Prelude", "Float"},
	"float64":  {"Prelude", "Double"},
	"bool":     {"Prelude", "Bool"},
	"bytes":    {"Data.ByteString", "ByteString"},
	"uuid":     {"Data.Text", "Text"},
	"instant":  {"Data.Time.Clock", "UTCTime"},
	"date":     {"Data.Time.Calendar", "Day"},
	"duration": {"Data.Time.Clock", "NominalDiffTime"},
	"decimal":  {"Prelude", "Rational"},
}

var (
	maybe   = imported{"Prelude", "Maybe"}
	set     = imported{"Data.Set", "Set"}
	mapping = imported{"Data.Map.Strict", "Map"}
)

// hsType is a rendered type. An atom needs no parentheses as an argument.
type hsType struct {
	text string
	atom bool
}

// arg is the type as an argument to a type constructor.
func (t hsType) arg() string {
	if t.atom {
		return t.text
	}
	return "(" + t.text + ")"
}

// Every reference to an imported name is written as a placeholder until
// the whole module is rendered, since whether an import is qualified
// depends on every name the module declares.
const (
	refStart = "\x00"
	refSep   = "\x01"
	refEnd   = "\x02"
)

var placeholder = regexp.MustCompile(refStart + `([^` + refSep + `]*)` + refSep + `([^` + refEnd + `]*)` + refEnd)

// use records an import and returns the placeholder for it.
func (g *generator) use(i imported) string {
	if g.uses[i.module] == nil {
		g.uses[i.module] = map[string]bool{}
	}
	g.uses[i.module][i.name] = true
	return refStart + i.module + refSep + i.name + refEnd
}

// typ renders a type reference. It walks the type itself rather than
// through [emit.Session.Resolve], since an alias the model declares keeps
// its name.
func (g *generator) typ(id *ir.ID) (hsType, error) {
	t := g.Model.Type(id)
	if t == nil {
		return hsType{}, emit.Unsupported(nil, "type %s did not resolve", id.GetName())
	}
	pos := t.GetPosition()

	switch {
	case t.GetParam() != nil:
		return hsType{}, emit.Lost(emit.LossGeneric, pos, "type parameter %s: generics are not generated yet", t.GetParam().GetName())
	case t.GetUnit() != nil:
		return hsType{}, emit.Lost(emit.LossUnit, pos, "a unit-typed field has no Haskell type yet")
	case t.GetExtern() != nil:
		return hsType{}, emit.Unsupported(pos, "%s is declared in another package, and foreign types are not generated yet", t.GetExtern().GetName())
	}

	decl := g.Model.Decl(t.GetCtor())
	if decl == nil {
		return hsType{}, emit.Unsupported(pos, "type %s did not resolve", t.GetCtor().GetName())
	}
	name := decl.GetMeta().GetName()
	args := t.GetArgs()
	arg := func(i int) (hsType, error) {
		if i >= len(args) {
			return hsType{}, emit.Unsupported(pos, "%s is missing a type argument", name)
		}
		return g.typ(args[i])
	}

	if a := decl.GetAlias(); a != nil {
		if len(a.GetParams()) > 0 {
			return hsType{}, emit.Lost(emit.LossGeneric, pos, "%s is parameterized, and generics are not generated yet", name)
		}
		if !emit.IsOwn(decl) {
			return g.typ(a.GetTarget())
		}
		// The synonym is declared, so its target must render.
		if _, err := g.typ(a.GetTarget()); err != nil {
			return hsType{}, err
		}
		return hsType{text: g.DeclName(decl, upperFirst), atom: true}, nil
	}

	if decl.GetPrimitive() != nil {
		switch name {
		case "List":
			elem, err := arg(0)
			if err != nil {
				return hsType{}, err
			}
			return hsType{text: "[" + elem.text + "]", atom: true}, nil
		case "Set":
			elem, err := arg(0)
			if err != nil {
				return hsType{}, err
			}
			return hsType{text: g.use(set) + " " + elem.arg()}, nil
		case "Map":
			key, err := arg(0)
			if err != nil {
				return hsType{}, err
			}
			val, err := arg(1)
			if err != nil {
				return hsType{}, err
			}
			return hsType{text: g.use(mapping) + " " + key.arg() + " " + val.arg()}, nil
		}
		s, ok := scalars[name]
		if !ok {
			return hsType{}, emit.Lost(emit.LossPrimitive, pos, "primitive %s has no Haskell type", name)
		}
		return hsType{text: g.use(s), atom: true}, nil
	}

	// SyntacticForm is deliberately not read: T? and T | null are both Maybe.
	if decl.GetEnumeration() != nil && (name == "Option" || name == "Nullable") && len(args) == 1 {
		elem, err := arg(0)
		if err != nil {
			return hsType{}, err
		}
		return hsType{text: g.use(maybe) + " " + elem.arg()}, nil
	}

	switch {
	case decl.GetClass() != nil:
		return hsType{}, emit.Unsupported(pos, "%s is a class, and a class is not a Haskell type", name)
	case decl.GetUnit() != nil:
		return hsType{}, emit.Lost(emit.LossUnit, pos, "%s is a unit, and units are not generated yet", name)
	case len(args) > 0:
		return hsType{}, emit.Lost(emit.LossGeneric, pos, "%s is applied to type arguments, and generics are not generated yet", name)
	}
	return hsType{text: g.DeclName(decl, upperFirst), atom: true}, nil
}

// imports writes the module's import declarations and replaces each
// placeholder in body. An import is qualified when the module declares a
// name it would bring in, aliased by its last segment, or its last two when
// another import holds that alias.
func (g *generator) imports(body string, declared map[string]bool) (string, string) {
	modules := make([]string, 0, len(g.uses))
	for m := range g.uses {
		modules = append(modules, m)
	}
	slices.Sort(modules)

	aliases := map[string]string{}
	taken := map[string]bool{}
	var b strings.Builder
	for _, m := range modules {
		names := make([]string, 0, len(g.uses[m]))
		qualified := false
		for n := range g.uses[m] {
			names = append(names, n)
			qualified = qualified || declared[n]
		}
		slices.Sort(names)

		if !qualified {
			b.WriteString("import " + m + " (" + strings.Join(names, ", ") + ")\n")
			continue
		}
		segs := strings.Split(m, ".")
		alias := segs[len(segs)-1]
		if taken[alias] && len(segs) > 1 {
			alias = strings.Join(segs[len(segs)-2:], ".")
		}
		taken[alias] = true
		aliases[m] = alias
		b.WriteString("import " + m + " qualified as " + alias + "\n")
	}

	body = placeholder.ReplaceAllStringFunc(body, func(ref string) string {
		sub := placeholder.FindStringSubmatch(ref)
		if alias, ok := aliases[sub[1]]; ok {
			return alias + "." + sub[2]
		}
		return sub[2]
	})
	return b.String(), body
}
