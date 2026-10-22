// Package zod generates a TypeScript module of Zod schemas from a resolved
// model: a schema parsing each declaration's JSON wire value, and a type
// inferred from it that is the typescript backend's type for the same
// declaration. docs/design/zod-backend.md is the design.
package zod

import (
	"cmp"
	"context"
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"

	"github.com/unstoppablemango/tdl/backend/internal/emit"
	"github.com/unstoppablemango/tdl/backend/internal/ts"
	"github.com/unstoppablemango/tdl/ir"
	"github.com/unstoppablemango/tdl/plugin"
)

// Name names this backend in a target block and as tdl-gen-zod.
const Name = "zod"

// Backend implements [plugin.Backend].
type Backend struct{}

func (Backend) Describe() plugin.Description {
	str := []ir.LiteralKind{ir.LiteralKind_LITERAL_KIND_STRING}
	return plugin.Description{
		Name:    Name,
		Version: "0.1.0",
		// Each request stands alone.
		Reuse: true,
		Directives: []*plugin.DirectiveSpec{
			// The TypeScript name for a schema, property, or variant schema.
			{Name: "name", MinArgs: 1, MaxArgs: 1, ArgKinds: str, Placements: plugin.OnMembers()},
			// The property a discriminated union switches on: the default on
			// the target block, an override on an enum.
			{Name: "discriminant", MinArgs: 1, MaxArgs: 1, ArgKinds: str, Placements: plugin.On(plugin.OnBlock, plugin.OnDecl)},
			// Refuses properties a struct or variant does not declare. On the
			// target block it applies to every one.
			{Name: "closed", Placements: plugin.On(plugin.OnBlock, plugin.OnDecl)},
			// A module and the schema it exports, for a declaration or an
			// extern another module owns: imported, and not declared.
			{Name: "foreign", MinArgs: 2, MaxArgs: 2, ArgKinds: []ir.LiteralKind{str[0], str[0]}, Placements: plugin.On(plugin.OnDecl)},
		},
	}
}

// Suffix ends the name of the file generated for a model, so a zod and a
// typescript target block can share an out directory.
const Suffix = ".zod.ts"

// A helper is a schema or function the module declares for itself, once,
// when a declaration uses it.
type helper int

const (
	helperDecimal helper = iota
	helperInstant
	helperUnique
	helpers
)

// helperNames is each helper's name, which no declaration may take.
var helperNames = [helpers]string{"decimal", "instant", "unique"}

// helperText is each helper's declaration. An instant is RFC 3339 with an
// offset, and Zod's pattern lets a time without seconds through, so the
// second pattern requires them. A set's elements are compared by a
// canonical JSON encoding with sorted keys, since two equal objects are
// distinct references.
var helperText = [helpers]string{
	`const decimal = z.string().regex(/^-?[0-9]+(\.[0-9]+)?$/);
`,
	`const instant = z.iso.datetime({ offset: true }).regex(/^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}/);
`,
	`const unique = (values: readonly unknown[]): boolean => {
  const canonical = (v: unknown): string => {
    if (Array.isArray(v)) {
      return "[" + v.map(canonical).join(",") + "]";
    }
    if (v !== null && typeof v === "object") {
      const o = v as Record<string, unknown>;
      return "{" + Object.keys(o).sort().map((k) => JSON.stringify(k) + ":" + canonical(o[k])).join(",") + "}";
    }
    return JSON.stringify(v);
  };
  const seen = new Set<string>();
  for (const v of values) {
    const key = canonical(v);
    if (seen.has(key)) {
      return false;
    }
    seen.add(key);
  }
  return true;
};
`,
}

// moduleNames are the names the module declares for itself, which no
// declaration may take: the import, the helpers, and the helpers later
// phases add.
var moduleNames = map[string]bool{"z": true, "codePoints": true, "size": true}

func init() {
	for _, n := range helperNames {
		moduleNames[n] = true
	}
}

// reserved refuses an import binding a name the module declares for
// itself, and takes the import back.
func (g *generator) reserved(name string, at *ir.Position) error {
	if !moduleNames[name] {
		return nil
	}
	delete(g.uses, name)
	return emit.Unsupported(at, "an import would bind %s, which the module declares for itself", name)
}

// schema is a Zod expression and the TypeScript type of the schema it
// builds, which a getter's return type spells.
type schema struct {
	expr, typ string
	// nullable is whether the schema already accepts null.
	nullable bool
}

// prims maps a TDL primitive to the schema checking its JSON value, which
// infers the type [ts.Scalars] gives it. A primitive's format is part of
// what it is, so it is checked and not branded.
var prims = map[string]schema{
	"string":   {expr: "z.string()", typ: "z.ZodString"},
	"int":      {expr: "z.int()", typ: "z.ZodInt"},
	"int32":    {expr: "z.int32()", typ: "z.ZodInt32"},
	"int64":    {expr: "z.int()", typ: "z.ZodInt"},
	"uint32":   {expr: "z.uint32()", typ: "z.ZodUInt32"},
	"uint64":   {expr: "z.int().nonnegative()", typ: "z.ZodInt"},
	"float32":  {expr: "z.float32()", typ: "z.ZodFloat32"},
	"float64":  {expr: "z.number()", typ: "z.ZodNumber"},
	"bool":     {expr: "z.boolean()", typ: "z.ZodBoolean"},
	"bytes":    {expr: "z.base64()", typ: "z.ZodBase64"},
	"decimal":  {expr: "decimal", typ: "typeof decimal"},
	"uuid":     {expr: "z.uuid()", typ: "z.ZodUUID"},
	"instant":  {expr: "instant", typ: "typeof instant"},
	"date":     {expr: "z.iso.date()", typ: "z.ZodISODate"},
	"duration": {expr: "z.iso.duration()", typ: "z.ZodISODuration"},
}

// primHelpers is the helper a primitive's schema names.
var primHelpers = map[string]helper{"decimal": helperDecimal, "instant": helperInstant}

type generator struct {
	*emit.Session

	// closed is whether the target block closes every object.
	closed bool

	// names maps each declared name to the declaration declaring it.
	names map[string]string

	// order is each declaration's place in source order, and current the
	// place of the one being rendered; a reference to one at or after it
	// is read only when parsing, since a const cannot be read before it
	// is initialized.
	order   map[*ir.Decl]int
	current int
	forward bool

	// lazy holds each newtype whose schema is lazy. A field naming one is
	// a getter too, since its type is written rather than inferred, and
	// TypeScript reads a getter's type only when it needs it.
	lazy map[*ir.Decl]bool

	// helpers holds the helpers the declaration being rendered names, and
	// uses what it imports.
	helpers [helpers]bool
	uses    ts.Uses

	// declares holds the names each rendered declaration declares.
	declares map[*ir.Decl][]string
}

// Generate returns one .zod.ts module holding the model's schemas.
func (Backend) Generate(_ context.Context, req *plugin.Request) (*plugin.Response, error) {
	g := &generator{
		Session:  emit.NewSession(req, "Zod"),
		names:    map[string]string{},
		order:    map[*ir.Decl]int{},
		lazy:     map[*ir.Decl]bool{},
		declares: map[*ir.Decl][]string{},
	}
	g.Externs = true
	g.closed = g.Bare("closed")

	path := ts.File(req.GetModel().GetPackage()) + Suffix

	own := g.Own()
	for i, d := range own {
		g.order[d] = i
	}
	texts := map[*ir.Decl]string{}
	helped := map[*ir.Decl][helpers]bool{}
	uses := map[*ir.Decl]ts.Uses{}
	skipped := map[*ir.Decl]bool{}
	// Each declaration's warnings are held until the cascade decides
	// whether it is emitted, since a skipped one loses everything at once.
	held := map[*ir.Decl][]*plugin.Diagnostic{}
	for i, d := range own {
		mark := len(g.Diags)
		g.current, g.helpers, g.uses = i, [helpers]bool{}, ts.Uses{}
		text, err := g.decl(d)
		uses[d] = g.uses
		if err != nil {
			g.Diags = g.Diags[:mark]
			g.Warn(err)
			skipped[d] = true
			continue
		}
		held[d] = slices.Clone(g.Diags[mark:])
		g.Diags = g.Diags[:mark]
		texts[d], helped[d] = text, g.helpers
	}
	for {
		g.Cascade(own, skipped)
		if !ts.Conflicts(own, skipped, g.declares, uses, g.Warn) {
			break
		}
	}

	var blocks []string
	var used [helpers]bool
	imports := ts.Uses{}
	for _, d := range own {
		if skipped[d] {
			continue
		}
		g.Diags = append(g.Diags, held[d]...)
		if texts[d] == "" {
			continue
		}
		blocks = append(blocks, texts[d])
		for h, u := range helped[d] {
			used[h] = used[h] || u
		}
		maps.Copy(imports, uses[d])
	}
	if len(blocks) == 0 {
		return g.Response(nil), nil
	}

	var b strings.Builder
	b.WriteString("// Code generated by tdl. DO NOT EDIT.\n\n")
	b.WriteString("import * as z from \"zod\";\n")
	b.WriteString(ts.Lines(imports, "import"))
	for h, u := range used {
		if u {
			b.WriteString("\n")
			b.WriteString(helperText[h])
		}
	}
	for _, block := range blocks {
		b.WriteString("\n")
		b.WriteString(block)
	}
	return g.Response([]*plugin.File{{Path: path, Content: []byte(b.String())}}), nil
}

// declares reports whether a declaration has a schema, or why it cannot
// have one yet. An alias is expanded where it is used and a primitive is a
// schema of its own, so neither declares anything.
func declares(d *ir.Decl) (bool, error) {
	pos := d.GetMeta().GetPosition()
	name := d.GetMeta().GetName()

	switch {
	case d.GetClass() != nil:
		return false, emit.Lost(emit.LossClass, pos, "%s is a class, and classes are not generated yet", name)
	case d.GetUnit() != nil:
		return false, emit.Lost(emit.LossUnit, pos, "%s is a unit, and units are not generated yet", name)
	case d.GetStructure() == nil && d.GetEnumeration() == nil && d.GetNewtype() == nil:
		return false, nil
	}
	if len(d.Params()) > 0 {
		return false, emit.Lost(emit.LossGeneric, pos, "%s is parameterized, and generics are not generated yet", name)
	}
	return true, nil
}

// decl renders one declaration, or "" for one that declares nothing.
func (g *generator) decl(d *ir.Decl) (string, error) {
	pos := d.GetMeta().GetPosition()
	name := d.GetMeta().GetName()

	if ok, err := declares(d); !ok {
		return "", err
	}
	if _, ok := g.Find(d.GetDirectives(), "foreign"); ok {
		g.foreignLosses(d)
		return "", nil
	}

	var b strings.Builder
	var declared []string
	var err error
	switch e := d.GetEnumeration(); {
	case d.GetNewtype() != nil:
		declared, err = g.newtype(&b, d)
	case d.GetStructure() != nil:
		closed := g.closed || g.Tagged(d.GetDirectives(), "closed")
		declared, err = g.object(&b, g.DeclName(d, emit.Pascal), d.GetMeta(), "", "", d.Fields(), closed)
	case emit.Fielded(e):
		declared, err = g.union(&b, d)
	default:
		declared = g.enum(&b, d)
	}
	if err != nil {
		return "", err
	}

	local := map[string]bool{}
	for _, n := range declared {
		if !ts.CanName(n) {
			return "", emit.Unsupported(pos, "%s would be named %s in TypeScript, which cannot name a type", name, n)
		}
		if moduleNames[n] {
			return "", emit.Unsupported(pos, "%s would be named %s, which the module declares for itself, and a name directive can rename it", name, n)
		}
		if local[n] {
			return "", emit.Unsupported(pos, "%s would declare %s in TypeScript twice", name, n)
		}
		local[n] = true
		if other, ok := g.names[n]; ok {
			return "", emit.Unsupported(pos, "%s would declare %s in TypeScript, and %s already does", name, n, other)
		}
	}
	for _, n := range declared {
		g.names[n] = name
	}
	g.declares[d] = declared

	g.WarnConstraints(d)
	g.LoseConforms(d, "Zod")
	g.unitLosses(d)
	return b.String(), nil
}

// allFields is a declaration's fields, its variants' included.
func (g *generator) allFields(d *ir.Decl) []*ir.Field {
	fields := slices.Clone(d.Fields())
	for _, v := range d.GetEnumeration().GetVariants() {
		fields = append(fields, v.GetFields()...)
	}
	return fields
}

// unitLosses warns about each field, and a newtype's base, measured in a
// unit, which is written as its base type until units are generated.
func (g *generator) unitLosses(d *ir.Decl) {
	name := emit.LastSegment(d.GetMeta().GetName())
	if nt := d.GetNewtype(); nt != nil && g.hasUnit(nt.GetBase(), map[int32]bool{}) {
		g.Lossy(emit.LossUnit, d.GetMeta().GetPosition(),
			"%s is measured in a unit, and Zod writes it as its base type until units are generated", name)
	}
	for _, f := range g.allFields(d) {
		if f.GetIncludedFrom() == nil && g.hasUnit(f.GetType(), map[int32]bool{}) {
			g.Lossy(emit.LossUnit, f.GetMeta().GetPosition(),
				"%s.%s is measured in a unit, and Zod writes it as its base type until units are generated", name, f.GetMeta().GetName())
		}
	}
}

// hasUnit reports whether a type applies a unit anywhere in it, through
// the aliases it names.
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

// pair writes a schema and the type inferred from it, each under the
// node's doc comment.
func pair(b *strings.Builder, name string, meta *ir.Meta, annotation, expr string) {
	emit.DocComment(b, "", meta)
	if annotation != "" {
		annotation = ": " + annotation
	}
	fmt.Fprintf(b, "export const %s%s = %s;\n", name, annotation, expr)
	emit.DocComment(b, "", meta)
	fmt.Fprintf(b, "export type %s = z.infer<typeof %s>;\n", name, name)
}

// newtype renders a newtype as the schema of its base. A base naming a
// declaration not yet initialized is lazy, with its type written so a
// cycle through it still infers.
func (g *generator) newtype(b *strings.Builder, d *ir.Decl) ([]string, error) {
	ref, err := g.Resolve(d.GetNewtype().GetBase())
	if err != nil {
		return nil, err
	}
	g.forward = false
	base, err := g.typ(ref)
	if err != nil {
		return nil, err
	}
	name := g.DeclName(d, emit.Pascal)
	if g.forward {
		g.lazy[d] = true
		pair(b, name, d.GetMeta(), "z.ZodLazy<"+base.typ+">", "z.lazy(() => "+base.expr+")")
	} else {
		pair(b, name, d.GetMeta(), "", base.expr)
	}
	return []string{name}, nil
}

// object renders an entity, a value, a mixin, or a variant as a
// z.object, or a z.strictObject when closed. A variant passes the
// discriminant and its value, which come first.
func (g *generator) object(b *strings.Builder, name string, meta *ir.Meta, disc, tag string, fields []*ir.Field, closed bool) ([]string, error) {
	var body strings.Builder
	seen := map[string]bool{}
	if disc != "" {
		seen[disc] = true
		fmt.Fprintf(&body, "  %s: z.literal(%s),\n", ts.Property(disc), strconv.Quote(tag))
	}

	for _, f := range fields {
		prop := g.FieldName(f, asWritten)
		pos := f.GetMeta().GetPosition()
		if seen[prop] {
			if prop == disc {
				return nil, emit.Unsupported(pos,
					"%s.%s takes the name of the discriminant, and a discriminant directive can rename it", name, prop)
			}
			return nil, emit.Unsupported(pos, "%s has two properties named %s in TypeScript", name, prop)
		}
		if prop == "__proto__" {
			return nil, emit.Unsupported(pos, "%s.%s sets an object's prototype in JavaScript, and a name directive can rename it", name, prop)
		}
		seen[prop] = true

		ref, err := g.Resolve(f.GetType())
		if err != nil {
			return nil, err
		}
		g.forward = false
		s, err := g.property(ref)
		if err != nil {
			return nil, err
		}
		if def := f.GetDefaultValue(); def != nil {
			s = g.withDefault(name, f, ref, s)
		}
		emit.DocComment(&body, "  ", f.GetMeta())
		if g.forward {
			fmt.Fprintf(&body, "  get %s(): %s {\n    return %s;\n  },\n", ts.Property(prop), s.typ, s.expr)
		} else {
			fmt.Fprintf(&body, "  %s: %s,\n", ts.Property(prop), s.expr)
		}
	}

	ctor := "z.object"
	if closed {
		ctor = "z.strictObject"
	}
	expr := ctor + "({})"
	if body.Len() > 0 {
		expr = ctor + "({\n" + body.String() + "})"
	}
	pair(b, name, meta, "", expr)
	return []string{name}, nil
}

// enum renders a fieldless enum as a z.enum of its tags, the union of
// string literals typescript writes. An enum with no variants holds no
// value.
func (g *generator) enum(b *strings.Builder, d *ir.Decl) []string {
	name := g.DeclName(d, emit.Pascal)
	variants := d.GetEnumeration().GetVariants()
	if len(variants) == 0 {
		pair(b, name, d.GetMeta(), "", "z.never()")
		return []string{name}
	}
	tags := make([]string, len(variants))
	for i, v := range variants {
		tags[i] = strconv.Quote(emit.Tag(v))
	}
	pair(b, name, d.GetMeta(), "", "z.enum(["+strings.Join(tags, ", ")+"])")
	return []string{name}
}

// union renders an enum where any variant carries fields: an object per
// variant, each with the discriminant set to its tag, and a
// z.discriminatedUnion of them.
func (g *generator) union(b *strings.Builder, d *ir.Decl) ([]string, error) {
	name := g.DeclName(d, emit.Pascal)
	disc := g.Discriminant(d)
	closed := g.closed || g.Tagged(d.GetDirectives(), "closed")

	declared := []string{name}
	var members []string
	for _, v := range d.GetEnumeration().GetVariants() {
		member := g.VariantName(v, func(s string) string { return name + emit.Pascal(s) })
		if _, err := g.object(b, member, v.GetMeta(), disc, emit.Tag(v), v.GetFields(), closed); err != nil {
			return nil, err
		}
		b.WriteString("\n")
		members = append(members, member)
		declared = append(declared, member)
	}
	// The union's type is written, since one variant naming the enum
	// would otherwise make it infer from itself.
	types := make([]string, len(members))
	for i, m := range members {
		types[i] = "typeof " + m
	}
	pair(b, name, d.GetMeta(),
		fmt.Sprintf("z.ZodDiscriminatedUnion<[%s], %s>", strings.Join(types, ", "), strconv.Quote(disc)),
		fmt.Sprintf("z.discriminatedUnion(%s, [%s])", strconv.Quote(disc), strings.Join(members, ", ")))
	return declared, nil
}

// property is a field's schema: exactOptional when the key may be absent,
// and nullable when the value may be null.
func (g *generator) property(r *emit.Ref) (schema, error) {
	optional, nullable := false, false
	for r.Form == emit.Option || r.Form == emit.Nullable {
		if r.Form == emit.Option {
			optional = true
		} else {
			nullable = true
		}
		r = r.Elem
	}
	s, err := g.typ(r)
	if err != nil {
		return schema{}, err
	}
	if nullable {
		s = orNull(s)
	}
	if optional {
		s = schema{expr: s.expr + ".exactOptional()", typ: "z.ZodExactOptional<" + s.typ + ">", nullable: s.nullable}
	}
	return s, nil
}

// orNull is a schema that also accepts null.
func orNull(s schema) schema {
	if s.nullable {
		return s
	}
	return schema{expr: s.expr + ".nullable()", typ: "z.ZodNullable<" + s.typ + ">", nullable: true}
}

// typ is the schema for a reference. An optional collection element may be
// null, since JSON has no absent element.
func (g *generator) typ(r *emit.Ref) (schema, error) {
	switch r.Form {
	case emit.Prim:
		s, ok := prims[r.Name]
		if !ok {
			return schema{}, emit.Unsupported(r.Pos, "primitive %s has no Zod schema", r.Name)
		}
		if h, ok := primHelpers[r.Name]; ok {
			g.helpers[h] = true
		}
		return s, nil
	case emit.List, emit.Set:
		elem, err := g.typ(r.Elem)
		if err != nil {
			return schema{}, err
		}
		s := schema{expr: "z.array(" + elem.expr + ")", typ: "z.ZodArray<" + elem.typ + ">"}
		if r.Form == emit.Set {
			g.helpers[helperUnique] = true
			s.expr += `.refine(unique, "a set holds each element once")`
		}
		return s, nil
	case emit.Map:
		return g.record(r)
	case emit.Option, emit.Nullable:
		inner, err := g.typ(r.Elem)
		if err != nil {
			return schema{}, err
		}
		return orNull(inner), nil
	case emit.Extern:
		name, err := ts.Extern(g.Session, g.uses, r, ".zod.js")
		if err != nil {
			return schema{}, err
		}
		if err := g.reserved(name, r.Pos); err != nil {
			return schema{}, err
		}
		return schema{expr: name, typ: "typeof " + name}, nil
	case emit.Named:
		if f, ok := g.Find(r.Decl.GetDirectives(), "foreign"); ok {
			// An import is initialized before the module runs, so no
			// getter is needed.
			args := f.GetArgs()
			name, err := g.uses.Add(r.Decl.GetMeta().GetName(), args[0].GetText(), args[1].GetText(), f.GetPosition())
			if err != nil {
				return schema{}, err
			}
			if err := g.reserved(name, f.GetPosition()); err != nil {
				return schema{}, err
			}
			return schema{expr: name, typ: "typeof " + name}, nil
		}
		if g.order[r.Decl] >= g.current || g.lazy[r.Decl] {
			g.forward = true
		}
		name := g.DeclName(r.Decl, emit.Pascal)
		return schema{expr: name, typ: "typeof " + name}, nil
	}
	return schema{}, emit.Unsupported(r.Pos, "%s has no Zod schema", cmp.Or(r.Name, "this type"))
}

// record renders a map as a z.record keyed by a string or a number, which
// Zod reads from a JSON key's digits, or a z.partialRecord keyed by a
// fieldless enum, since a z.record keyed by one would demand every key.
func (g *generator) record(r *emit.Ref) (schema, error) {
	kx, err := g.Expand(r.Key)
	if err != nil {
		return schema{}, err
	}
	key, err := g.typ(r.Key)
	if err != nil {
		return schema{}, err
	}
	val, err := g.typ(r.Elem)
	if err != nil {
		return schema{}, err
	}

	switch {
	case kx.Form == emit.Prim && (ts.Scalars[kx.Name] == "string" || ts.Scalars[kx.Name] == "number"):
		return schema{
			expr: "z.record(" + key.expr + ", " + val.expr + ")",
			typ:  "z.ZodRecord<" + key.typ + ", " + val.typ + ">",
		}, nil
	case kx.Form == emit.Named && kx.Decl.GetEnumeration() != nil && !emit.Fielded(kx.Decl.GetEnumeration()):
		return schema{
			expr: "z.partialRecord(" + key.expr + ", " + val.expr + ")",
			typ:  "z.ZodRecord<" + key.typ + " & z.core.$partial, " + val.typ + ">",
		}, nil
	}
	return schema{}, emit.Unsupported(r.Pos, "a Zod record key is a string, a number, or a fieldless enum, and this one is %s", key.expr)
}

func asWritten(name string) string { return name }
