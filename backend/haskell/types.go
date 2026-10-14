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
	if i.module == g.self {
		// A mapping onto the generated module itself imports nothing.
		return i.name
	}
	if g.uses[i.module] == nil {
		g.uses[i.module] = map[string]bool{}
	}
	g.uses[i.module][i.name] = true
	return refStart + i.module + refSep + i.name + refEnd
}

// typ renders a type reference. It walks the type itself rather than
// through [emit.Session.Resolve], since an alias the model declares keeps
// its name and a type parameter is a type variable.
func (g *generator) typ(id *ir.ID) (hsType, error) { return g.render(bound{id, nil}) }

// render renders a type reference in a frame.
func (g *generator) render(b bound) (hsType, error) {
	a, err := g.resolve(b)
	if err != nil {
		return hsType{}, err
	}
	t, args := a.t, a.args
	pos := t.GetPosition()

	switch {
	case t.GetParam() != nil:
		return g.apply(hsType{text: tyvar(t.GetParam().GetName()), atom: true}, args)
	case t.GetUnit() != nil:
		return hsType{}, emit.Lost(emit.LossUnit, pos, "a unit-typed field has no Haskell type yet")
	case t.GetExtern() != nil:
		f, ok := g.extern(t)
		if !ok {
			return hsType{}, emit.Unsupported(pos,
				"%s is declared in another package, which has no haskell target block, and no foreign directive maps it", t.GetExtern().GetName())
		}
		return g.applied(f, args)
	}

	decl := g.Model.Decl(t.GetCtor())
	if decl == nil {
		return hsType{}, emit.Unsupported(pos, "type %s did not resolve", t.GetCtor().GetName())
	}
	if f, ok := g.foreign[decl]; ok {
		return g.applied(f, args)
	}
	name := decl.GetMeta().GetName()
	most := func(n int) error {
		if len(args) > n {
			return emit.Unsupported(pos, "%s is applied to %d type arguments, and takes %d", name, len(args), n)
		}
		return nil
	}

	if a := decl.GetAlias(); a != nil {
		n := len(a.GetParams())
		if len(args) < n {
			return hsType{}, emit.Unsupported(pos, "%s is a synonym applied to %d of its %d type arguments, which Haskell refuses", name, len(args), n)
		}
		target := bound{a.GetTarget(), &frame{args[:n]}}
		if !emit.IsOwn(decl) {
			if err := most(n); err != nil {
				return hsType{}, err
			}
			return g.render(target)
		}
		// The synonym is declared, so its target must render.
		if _, err := g.render(target); err != nil {
			return hsType{}, err
		}
		return g.apply(hsType{text: g.DeclName(decl, upperFirst), atom: true}, args)
	}

	if decl.GetPrimitive() != nil {
		switch name {
		case "List":
			if err := most(1); err != nil {
				return hsType{}, err
			}
			if len(args) == 0 {
				return hsType{text: "[]", atom: true}, nil
			}
			elem, err := g.render(args[0])
			if err != nil {
				return hsType{}, err
			}
			return hsType{text: "[" + elem.text + "]", atom: true}, nil
		case "Set":
			if err := most(1); err != nil {
				return hsType{}, err
			}
			return g.applied(set, args)
		case "Map":
			if err := most(2); err != nil {
				return hsType{}, err
			}
			return g.applied(mapping, args)
		}
		s, ok := scalars[name]
		if !ok {
			return hsType{}, emit.Lost(emit.LossPrimitive, pos, "primitive %s has no Haskell type", name)
		}
		if err := most(0); err != nil {
			return hsType{}, err
		}
		return hsType{text: g.use(s), atom: true}, nil
	}

	// SyntacticForm is deliberately not read: T? and T | null are both Maybe.
	if optional(decl) {
		if err := most(1); err != nil {
			return hsType{}, err
		}
		return g.applied(maybe, args)
	}

	switch {
	case decl.GetClass() != nil:
		return hsType{}, emit.Unsupported(pos, "%s is a class, and a class is not a Haskell type", name)
	case decl.GetUnit() != nil:
		return hsType{}, emit.Lost(emit.LossUnit, pos, "%s is a unit, and units are not generated yet", name)
	}
	return g.apply(hsType{text: g.DeclName(decl, upperFirst), atom: true}, args)
}

// apply is a type applied to type arguments.
func (g *generator) apply(fn hsType, args []bound) (hsType, error) {
	if len(args) == 0 {
		return fn, nil
	}
	text := fn.arg()
	for _, a := range args {
		t, err := g.render(a)
		if err != nil {
			return hsType{}, err
		}
		text += " " + t.arg()
	}
	return hsType{text: text}, nil
}

// applied is an imported type applied to type arguments.
func (g *generator) applied(f imported, args []bound) (hsType, error) {
	return g.apply(hsType{text: g.use(f), atom: true}, args)
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
