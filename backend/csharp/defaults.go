package csharp

import (
	"math"
	"strconv"
	"strings"

	"github.com/unstoppablemango/tdl/backend/internal/emit"
	"github.com/unstoppablemango/tdl/ir"
)

// defaultValue renders a field's default as the property's initializer. An
// error is a default C# cannot write, which the caller warns at and drops,
// leaving the property required.
func (g *generator) defaultValue(id *ir.ID, lit *ir.Literal, shadow map[string]bool) (string, error) {
	r, err := g.Resolve(id)
	if err != nil {
		return "", err
	}
	return g.literal(r, lit, shadow)
}

// literal renders lit as a value of the type r.
func (g *generator) literal(r *emit.Ref, lit *ir.Literal, shadow map[string]bool) (string, error) {
	pos := lit.GetPosition()
	kind := lit.GetKind()
	text := lit.GetText()
	drop := func(format string, args ...any) (string, error) {
		return "", emit.Lost(emit.LossDefault, pos, "the default %s is dropped: "+format, append([]any{emit.LiteralText(lit)}, args...)...)
	}

	switch r.Form {
	case emit.Option, emit.Nullable:
		return g.literal(r.Elem, lit, shadow)

	case emit.Extern:
		return drop("%s.%s is a foreign type, whose values another assembly decides", r.Extern.GetPackage(), r.Extern.GetName())

	case emit.Prim:
		if _, ok := g.foreignPrim(r.ID); ok {
			return drop("%s is a foreign type, whose values another assembly decides", r.Name)
		}
		switch r.Name {
		case "string":
			if kind == ir.LiteralKind_LITERAL_KIND_STRING {
				return str(text), nil
			}
		case "bool":
			if kind == ir.LiteralKind_LITERAL_KIND_BOOL {
				return text, nil
			}
		case "int", "int64", "int32", "uint32", "uint64":
			if kind == ir.LiteralKind_LITERAL_KIND_INT {
				if !fits(r.Name, text) {
					return drop("it is out of range for %s", r.Name)
				}
				return text, nil
			}
		case "float32", "float64", "decimal":
			if kind != ir.LiteralKind_LITERAL_KIND_INT && kind != ir.LiteralKind_LITERAL_KIND_FLOAT {
				break
			}
			switch r.Name {
			case "float32":
				return text + "f", nil
			case "decimal":
				return text + "m", nil
			}
			if kind == ir.LiteralKind_LITERAL_KIND_INT {
				return text + ".0", nil
			}
			return text, nil
		default:
			return drop("C# has no constant of type %s", r.Name)
		}

	case emit.List, emit.Set:
		if kind != ir.LiteralKind_LITERAL_KIND_LIST {
			break
		}
		elem, err := g.render(r.Elem, shadow)
		if err != nil {
			return "", err
		}
		factory := immutable + "ImmutableList"
		if r.Form == emit.Set {
			factory = immutable + "ImmutableHashSet"
		}
		if len(lit.GetItems()) == 0 {
			return factory + "<" + elem.text + ">.Empty", nil
		}
		items := make([]string, len(lit.GetItems()))
		for i, item := range lit.GetItems() {
			if items[i], err = g.literal(r.Elem, item, shadow); err != nil {
				return "", err
			}
		}
		return factory + ".Create<" + elem.text + ">(" + strings.Join(items, ", ") + ")", nil

	case emit.Map:
		if kind != ir.LiteralKind_LITERAL_KIND_LIST || len(lit.GetItems()) > 0 {
			break
		}
		t, err := g.render(r, shadow)
		if err != nil {
			return "", err
		}
		return immutable + "ImmutableDictionary" + strings.TrimPrefix(t.text, mapType) + ".Empty", nil

	case emit.Named:
		if _, ok := g.foreign[r.Decl]; ok {
			return drop("%s is a foreign type, whose values another assembly decides", r.Decl.GetMeta().GetName())
		}
		t, err := g.render(r, shadow)
		if err != nil {
			return "", err
		}
		d := r.Decl
		switch {
		case d.GetNewtype() != nil:
			base, err := g.Resolve(d.GetNewtype().GetBase())
			if err != nil {
				return "", err
			}
			// A newtype's base is written from the top level, where no
			// variant shadows a name.
			inner, err := g.literal(base, lit, nil)
			if err != nil {
				return "", err
			}
			return "new " + t.text + "(" + inner + ")", nil
		case d.GetEnumeration() != nil && kind == ir.LiteralKind_LITERAL_KIND_NAME:
			v := variantOf(d.GetEnumeration(), lit)
			if v == nil {
				return drop("%s has no variant %s", d.GetMeta().GetName(), text)
			}
			name, err := g.memberName(g.VariantName(v, upperFirst))
			if err != nil {
				return "", err
			}
			if !emit.Fielded(d.GetEnumeration()) {
				return t.text + "." + name, nil
			}
			if len(v.GetFields()) > 0 {
				return drop("variant %s has fields", v.GetMeta().GetName())
			}
			return "new " + t.text + "." + name + "()", nil
		}
		return drop("C# has no constant of type %s", d.GetMeta().GetName())
	}
	return drop("it is not a value of its field's type")
}

// variantOf is the variant a name denotes. Lowering resolves a default
// written as a name, but not a name inside a list.
func variantOf(e *ir.Enum, lit *ir.Literal) *ir.Variant {
	vs := e.GetVariants()
	if id := lit.GetVariant(); id != nil && int(id.GetIndex()) < len(vs) {
		return vs[id.GetIndex()]
	}
	for _, v := range vs {
		if v.GetMeta().GetName() == lit.GetText() {
			return v
		}
	}
	return nil
}

// fits reports whether an integer literal is in range for the primitive.
func fits(prim, text string) bool {
	if strings.HasPrefix(prim, "u") {
		n, err := strconv.ParseUint(text, 10, 64)
		return err == nil && (prim == "uint64" || n <= math.MaxUint32)
	}
	n, err := strconv.ParseInt(text, 10, 64)
	return err == nil && (prim != "int32" || n >= math.MinInt32 && n <= math.MaxInt32)
}
