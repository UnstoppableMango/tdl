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
	"uuid":     matching(`[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[1-8][0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}|00000000-0000-0000-0000-000000000000|ffffffff-ffff-ffff-ffff-ffffffffffff`),
	"date":     matching(date),
	"instant":  matching(date + `[Tt]` + hms + `(\.[0-9]+)?([Zz]|[+-]([01][0-9]|2[0-3]):[0-5][0-9])`),
	"duration": "duration",
}

// The wire forms of RFC 3339's full-date and time, as zod and JSON Schema
// read them.
const (
	date = `[0-9]{4}-(0[1-9]|1[0-2])-(0[1-9]|[12][0-9]|3[01])`
	hms  = `([01][0-9]|2[0-3]):[0-5][0-9]:([0-5][0-9]|60)`
)

// matching is a string type checked by a POSIX ERE, which builtins.match
// anchors at both ends.
func matching(ere string) string { return "types.strMatching " + str(ere) }

// afterHelper applies f to a type's merged value. nixpkgs merges through
// merge.v2 where it has one, and rebuilds a type holding submodules
// through substSubModules, which must keep the wrapper.
const afterHelper = `  after =
    f: base:
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
              r // { value = f args.loc r.value; };
          }
        else
          loc: defs: f loc (base.merge loc defs);
      substSubModules = m: after f (base.substSubModules m);
    };
`

// structHelper is a submodule whose optional fields, unset, are absent
// from its value, so toJSON leaves them out.
const structHelper = `  struct =
    options:
    let
      absent = n: v: v == null && options.${n}.optional or false;
    in
    after (_: lib.filterAttrs (n: v: !absent n v)) (
      types.submodule { options = lib.mapAttrs (_: o: removeAttrs o [ "optional" ]) options; }
    );
`

// opaqueHelper hides a struct's options from docs, for a struct reached
// from itself, whose docs would otherwise never end.
const opaqueHelper = `  opaque =
    type:
    type
    // {
      getSubOptions = _: { };
      substSubModules = m: opaque (type.substSubModules m);
    };
`

// durationHelper is an ISO 8601 duration. An ERE cannot look ahead, so a
// check refuses P, and a T ending it, which hold no component.
const durationHelper = `  duration =
    types.addCheck
      (types.strMatching "P([0-9]+W|([0-9]+Y)?([0-9]+M)?([0-9]+D)?(T([0-9]+H)?([0-9]+M)?([0-9]+([.,][0-9]+)?S)?)?)")
      (s: builtins.match "P|.*T" s == null);
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
	g.used["struct"], g.used["after"] = true, true
	g.current = d
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
		switch r.Name {
		case "duration":
			g.used["duration"] = true
		case "uint64":
			g.Lossy(emit.LossPrimitive, r.Pos, "uint64 is an unsigned Nix int, which stops at 2^63-1")
		}
		return t, nil
	case emit.List, emit.Set, emit.Option, emit.Nullable:
		elem, err := g.typ(r.Elem)
		if err != nil {
			return "", err
		}
		fn := map[emit.Form]string{emit.List: "types.listOf", emit.Set: "set", emit.Option: "types.nullOr", emit.Nullable: "types.nullOr"}[r.Form]
		if r.Form == emit.Set {
			g.used["set"], g.used["after"] = true, true
		}
		return fn + " " + arg(elem), nil
	case emit.Map:
		return g.attrs(r)
	}
	if r.Decl.GetStructure() != nil && g.reaches(r.Decl, g.current, map[*ir.Decl]bool{}) {
		g.used["opaque"] = true
		return "opaque " + g.attr(r.Decl), nil
	}
	return g.attr(r.Decl), nil
}

// reaches reports whether from names to, through any number of
// declarations.
func (g *generator) reaches(from, to *ir.Decl, seen map[*ir.Decl]bool) bool {
	if from == to {
		return true
	}
	if seen[from] {
		return false
	}
	seen[from] = true
	return slices.ContainsFunc(g.References(from), func(d *ir.Decl) bool { return g.reaches(d, to, seen) })
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
