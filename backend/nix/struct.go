package nix

import (
	"fmt"
	"slices"
	"strings"

	"github.com/unstoppablemango/tdl/backend/internal/emit"
	"github.com/unstoppablemango/tdl/ir"
)

// prims maps a primitive to its option type. Nix has one signed 64-bit
// integer, and a JSON 1 arrives as an int, so a float is a number. What
// JSON writes as a string is checked against its wire form.
var prims = map[string]string{
	"string":   "types.str",
	"bool":     "types.bool",
	"int":      "types.int",
	"int64":    "types.int",
	"int32":    "types.ints.s32",
	"uint32":   "types.ints.u32",
	"uint64":   "types.ints.unsigned",
	"float32":  "types.number",
	"float64":  "types.number",
	"bytes":    matching(`([A-Za-z0-9+/]{4})*([A-Za-z0-9+/]{2}==|[A-Za-z0-9+/]{3}=)?`),
	"decimal":  matching(`-?[0-9]+(\.[0-9]+)?`),
	"uuid":     matching(`[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}`),
	"date":     matching(`[0-9]{4}-[0-9]{2}-[0-9]{2}`),
	"instant":  matching(`[0-9]{4}-[0-9]{2}-[0-9]{2}[Tt][0-9]{2}:[0-9]{2}:[0-9]{2}(\.[0-9]+)?([Zz]|[+-][0-9]{2}:[0-9]{2})`),
	"duration": matching(`P([0-9]+W|([0-9]+Y)?([0-9]+M)?([0-9]+D)?(T([0-9]+H)?([0-9]+M)?([0-9]+([.,][0-9]+)?S)?)?)`),
}

// matching is a string type checked by a POSIX ERE, which builtins.match
// anchors at both ends.
func matching(ere string) string { return "types.strMatching " + str(ere) }

// structHelper is a submodule whose optional fields, unset, are absent
// from its value, so toJSON leaves them out. nixpkgs rebuilds a submodule
// through substSubModules, which must keep the wrapper.
const structHelper = `  struct =
    options:
    let
      absent = n: v: v == null && options.${n}.optional or false;
      drop = lib.filterAttrs (n: v: !absent n v);
      wrap =
        base:
        base
        // {
          merge =
            if base.merge ? v2 then
              base.merge
              // {
                v2 =
                  args:
                  let
                    r = base.merge.v2 args;
                  in
                  r // { value = drop r.value; };
              }
            else
              loc: defs: drop (base.merge loc defs);
          substSubModules = m: wrap (base.substSubModules m);
        };
    in
    wrap (types.submodule { options = lib.mapAttrs (_: o: removeAttrs o [ "optional" ]) options; });
`

// optionalHelper is an option that defaults to null, marked for struct.
const optionalHelper = `  optional =
    attrs:
    mkOption (
      attrs
      // {
        type = types.nullOr attrs.type;
        default = null;
      }
    )
    // {
      optional = true;
    };
`

// structure renders an entity, a value, or a mixin, which emit alike, as
// a submodule. A mixin's fields are already copied into each includer.
func (g *generator) structure(b *strings.Builder, d *ir.Decl, attr string) error {
	g.used["struct"] = true
	owner := emit.LastSegment(d.GetMeta().GetName())
	var fields strings.Builder
	seen := map[string]bool{}
	for _, f := range d.Fields() {
		if err := g.field(&fields, owner, f, seen); err != nil {
			return err
		}
	}
	if fields.Len() == 0 {
		fmt.Fprintf(b, "  %s = struct { };\n", attr)
	} else {
		fmt.Fprintf(b, "  %s = struct {\n%s  };\n", attr, fields.String())
	}
	return nil
}

// field renders one option of a struct.
func (g *generator) field(b *strings.Builder, owner string, f *ir.Field, seen map[string]bool) error {
	pos := f.GetMeta().GetPosition()
	attr := g.Property(f)
	g.WarnPropertyName(owner, f)
	if seen[attr] {
		return emit.Unsupported(pos, "%s has two attributes named %s", owner, attr)
	}
	seen[attr] = true

	r, err := g.Resolve(f.GetType())
	if err != nil {
		return err
	}
	optional, nullable := false, false
	for r.Form == emit.Option || r.Form == emit.Nullable {
		optional = optional || r.Form == emit.Option
		nullable = nullable || r.Form == emit.Nullable
		r = r.Elem
	}
	if optional && nullable {
		g.Lossy(emit.LossOptional, pos, "%s.%s is both optional and nullable, and is written as optional", owner, f.GetMeta().GetName())
	}
	t, err := g.typ(r)
	if err != nil {
		return err
	}
	if nullable && !optional {
		g.used["nullable"] = true
		t = "nullable " + arg(t)
	}
	if g.hasUnit(f.GetType(), map[int32]bool{}) {
		g.Lossy(emit.LossUnit, pos, "%s.%s is measured in a unit, which Nix does not carry", owner, f.GetMeta().GetName())
	}
	if f.GetDefaultValue() != nil {
		g.Lossy(emit.LossDefault, pos, "%s.%s's default is not generated yet", owner, f.GetMeta().GetName())
	}

	key := attr
	if !ident.MatchString(key) || slices.Contains(keywords, key) {
		key = str(key)
	}
	fn := "mkOption"
	if optional {
		fn = "optional"
		g.used[fn] = true
	}
	fmt.Fprintf(b, "    %s = %s {\n      type = %s;\n", key, fn, t)
	if doc := description(f.GetMeta()); doc != "" {
		fmt.Fprintf(b, "      description = %s;\n", str(doc))
	}
	b.WriteString("    };\n")
	return nil
}

// typ is the option type for a reference.
func (g *generator) typ(r *emit.Ref) (string, error) {
	switch r.Form {
	case emit.Prim:
		t, ok := prims[r.Name]
		if !ok {
			return "", emit.Unsupported(r.Pos, "primitive %s has no Nix type", r.Name)
		}
		if r.Name == "uint64" {
			g.Lossy(emit.LossPrimitive, r.Pos, "uint64 is an unsigned Nix int, which stops at 2^63-1")
		}
		return t, nil
	case emit.List, emit.Option, emit.Nullable:
		elem, err := g.typ(r.Elem)
		if err != nil {
			return "", err
		}
		fn := map[emit.Form]string{emit.List: "types.listOf", emit.Option: "types.nullOr", emit.Nullable: "types.nullOr"}[r.Form]
		return fn + " " + arg(elem), nil
	case emit.Set, emit.Map:
		return "", emit.Unsupported(r.Pos, "sets and maps are not generated yet")
	}
	return g.attr(r.Decl), nil
}

// hasUnit reports whether a type is measured in a unit.
func (g *generator) hasUnit(id *ir.ID, seen map[int32]bool) bool {
	t := g.Model.Type(id)
	if t == nil || seen[id.GetIndex()] {
		return false
	}
	seen[id.GetIndex()] = true
	if t.GetUnit() != nil {
		return true
	}
	if a := g.Model.Decl(t.GetCtor()).GetAlias(); a != nil && g.hasUnit(a.GetTarget(), seen) {
		return true
	}
	return slices.ContainsFunc(t.GetArgs(), func(arg *ir.ID) bool { return g.hasUnit(arg, seen) })
}

// arg parenthesizes an application passed as an argument.
func arg(t string) string {
	if strings.Contains(t, " ") {
		return "(" + t + ")"
	}
	return t
}
