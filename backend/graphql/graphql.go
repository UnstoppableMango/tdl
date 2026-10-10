// Package graphql generates a GraphQL schema of output types from a
// resolved model. The mapping is in docs/design/schema-backends.md.
package graphql

import (
	"cmp"
	"context"
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strings"

	"github.com/unstoppablemango/tdl/backend/internal/emit"
	"github.com/unstoppablemango/tdl/internal/unlower"
	"github.com/unstoppablemango/tdl/ir"
	"github.com/unstoppablemango/tdl/plugin"
)

// Name names this backend in a target block and as tdl-gen-graphql.
const Name = "graphql"

// Backend implements [plugin.Backend].
type Backend struct{}

func (Backend) Describe() plugin.Description {
	return plugin.Description{
		Name:    Name,
		Version: "0.1.0",
		// Each request stands alone.
		Reuse: true,
		Directives: []*plugin.DirectiveSpec{
			// The GraphQL name for a type, field, enum value, or variant
			// object type.
			{Name: "name", MinArgs: 1, MaxArgs: 1, ArgKinds: []ir.LiteralKind{ir.LiteralKind_LITERAL_KIND_STRING}},
			// Writes, as @tdl directives, every fact the schema alone would
			// lose, so import rebuilds the model.
			{Name: "roundtrip"},
		},
		Reverse: true,
	}
}

// scalars maps a TDL primitive to its GraphQL scalar. GraphQL's Int is
// 32-bit, so a wider integer is a custom scalar.
var scalars = map[string]string{
	"string":   "String",
	"int":      "Long",
	"int32":    "Int",
	"int64":    "Long",
	"uint32":   "Long",
	"uint64":   "UInt64",
	"float32":  "Float",
	"float64":  "Float",
	"bool":     "Boolean",
	"bytes":    "Bytes",
	"decimal":  "Decimal",
	"uuid":     "UUID",
	"instant":  "DateTime",
	"date":     "Date",
	"duration": "Duration",
}

// custom describes each custom scalar by meaning, since TDL defines no wire
// encoding.
var custom = map[string]string{
	"Long":     "A 64-bit signed integer.",
	"UInt64":   "A 64-bit unsigned integer.",
	"Bytes":    "Binary data.",
	"Decimal":  "An exact decimal number.",
	"UUID":     "A universally unique identifier.",
	"DateTime": "A point in time.",
	"Date":     "A calendar date.",
	"Duration": "A length of time.",
}

// builtin is the scalars GraphQL declares itself.
var builtin = map[string]bool{"String": true, "Int": true, "Float": true, "Boolean": true, "ID": true}

var ident = regexp.MustCompile(`^[_A-Za-z][_0-9A-Za-z]*$`)

// placeholder is the field a fieldless variant carries, since a GraphQL
// object needs one.
const placeholder = "_"

type generator struct {
	*emit.Session

	// local is every type name the model's declarations take, variant
	// object types included, for a custom scalar to collide with.
	local map[string]bool

	// names maps each declared type name to the declaration declaring it.
	names map[string]string

	// uses is the custom scalars the declaration being rendered needs.
	uses map[string]bool

	// annotated is whether any @tdl directive is written, which the schema
	// must then define.
	annotated bool
}

type rendered struct {
	decl *ir.Decl
	text string
	uses map[string]bool
}

// defaultFile names the file of a model without a package.
const defaultFile = "schema"

// Generate returns one .graphql file holding the model's declarations.
func (Backend) Generate(_ context.Context, req *plugin.Request) (*plugin.Response, error) {
	g := &generator{
		Session: emit.NewSession(req, "GraphQL"),
		local:   map[string]bool{},
		names:   map[string]string{},
	}
	g.Roundtrip = g.Bare("roundtrip")

	pkg := req.GetModel().GetPackage()
	if back := packageBack(pkg); back != pkg {
		g.Lose(emit.LossName, nil, "package %s names the file %s.graphql, which reads back as package %q", pkg, cmp.Or(back, defaultFile), back)
	}
	if len(req.GetModel().GetDoc()) > 0 {
		g.Lose(emit.LossDoc, nil, "GraphQL has no place for the package's doc comment")
	}

	own := g.Own()
	for _, d := range own {
		for _, n := range g.declares(d) {
			g.local[n] = true
		}
	}

	skipped := map[*ir.Decl]bool{}
	// Each declaration's warnings are held until the cascade decides
	// whether it is emitted, since a skipped one loses everything at once.
	held := map[*ir.Decl][]*plugin.Diagnostic{}
	var out []rendered
	for _, d := range own {
		g.uses = map[string]bool{}
		mark := len(g.Diags)
		text, err := g.decl(d)
		if err != nil {
			g.Diags = g.Diags[:mark]
			g.Skip(err)
			skipped[d] = true
			continue
		}
		held[d] = slices.Clone(g.Diags[mark:])
		g.Diags = g.Diags[:mark]
		if text != "" {
			out = append(out, rendered{d, text, g.uses})
		}
	}
	mark := len(g.Diags)
	g.Cascade(own, skipped)
	if g.Roundtrip {
		// A skipped declaration is carried as TDL, so nothing is lost.
		g.Diags = g.Diags[:mark]
	}
	for _, d := range own {
		if !skipped[d] {
			g.Diags = append(g.Diags, held[d]...)
		}
	}

	var blocks []string
	need := map[string]bool{}
	emitted := map[string]bool{}
	for _, r := range out {
		if skipped[r.decl] {
			continue
		}
		blocks = append(blocks, r.text)
		maps.Copy(need, r.uses)
		emitted[r.decl.GetMeta().GetName()] = true
	}
	header := g.schemaExtension(emitted)
	if len(blocks) == 0 && header == "" {
		return g.Response(nil), nil
	}
	for _, s := range slices.Sorted(maps.Keys(need)) {
		var b strings.Builder
		description(&b, "", []string{custom[s]})
		fmt.Fprintf(&b, "scalar %s\n", s)
		blocks = append(blocks, b.String())
	}
	if header != "" {
		blocks = append([]string{header}, blocks...)
	}
	if g.annotated {
		blocks = append([]string{definition}, blocks...)
	}

	var b strings.Builder
	b.WriteString("# Code generated by tdl. DO NOT EDIT.\n")
	for _, block := range blocks {
		b.WriteString("\n")
		b.WriteString(block)
	}

	path := defaultFile + ".graphql"
	if pkg != "" {
		path = emit.LastSegment(pkg) + ".graphql"
	}
	return g.Response([]*plugin.File{{Path: path, Content: []byte(b.String())}}), nil
}

// declares is every type name a declaration takes, read before rendering
// so a custom scalar sees the whole namespace.
func (g *generator) declares(d *ir.Decl) []string {
	if d.GetStructure() == nil && d.GetEnumeration() == nil {
		return nil
	}
	name := g.DeclName(d, emit.Pascal)
	e := d.GetEnumeration()
	if e == nil || !emit.Fielded(e) {
		return []string{name}
	}
	names := []string{name}
	for _, v := range e.GetVariants() {
		names = append(names, g.memberName(name, v))
	}
	return names
}

// memberName is the object type a variant of a union becomes.
func (g *generator) memberName(union string, v *ir.Variant) string {
	return g.VariantName(v, func(s string) string { return union + emit.Pascal(s) })
}

// decl renders one declaration, or "" for one that declares nothing.
func (g *generator) decl(d *ir.Decl) (string, error) {
	pos := d.GetMeta().GetPosition()
	name := d.GetMeta().GetName()

	switch {
	case d.GetClass() != nil:
		return "", emit.Lost(emit.LossClass, pos, "%s is a class, and classes are not generated yet", name)
	case d.GetUnit() != nil:
		return "", emit.Lost(emit.LossUnit, pos, "%s is a unit, and units are not generated yet", name)
	case d.GetAlias() != nil:
		if len(d.Params()) == 0 {
			g.Lose(emit.LossAlias, pos, "alias %s is expanded where used", name)
			g.typeLosses(d.GetAlias().GetTarget(), nil)
		}
		return "", nil
	case d.GetStructure() == nil && d.GetEnumeration() == nil && d.GetNewtype() == nil:
		// A primitive declares nothing.
		return "", nil
	}
	if len(d.Params()) > 0 {
		return "", emit.Lost(emit.LossGeneric, pos, "%s is parameterized, and generics are not generated yet", name)
	}

	if n := d.GetNewtype(); n != nil {
		// A newtype is expanded, since a custom scalar would need server code.
		ref, err := g.Resolve(n.GetBase())
		if err == nil {
			_, err = g.Expand(ref)
		}
		if err != nil {
			return "", err
		}
		g.Lose(emit.LossNewtype, pos, "newtype %s is expanded where used", name)
		g.typeLosses(n.GetBase(), nil)
		if !g.Roundtrip {
			g.WarnWhere(d)
		}
		return "", nil
	}

	var b strings.Builder
	var declared []string
	var err error
	switch e := d.GetEnumeration(); {
	case e == nil:
		written := g.DeclName(d, emit.Pascal)
		declared, err = g.object(&b, written, d.GetMeta(), d.Fields(), object, g.structKVs(d, written))
	case emit.Fielded(e):
		declared, err = g.union(&b, d)
	default:
		declared, err = g.enum(&b, d)
	}
	if err != nil {
		return "", err
	}

	seen := map[string]bool{}
	for _, n := range declared {
		if err := g.typeName(n, pos); err != nil {
			return "", err
		}
		if other, ok := g.names[n]; ok {
			return "", emit.Unsupported(pos, "%s would declare %s in GraphQL, and %s already does", name, n, other)
		}
		if seen[n] {
			return "", emit.Unsupported(pos, "%s would declare %s in GraphQL twice", name, n)
		}
		seen[n] = true
	}
	for _, n := range declared {
		g.names[n] = name
	}
	if !g.Roundtrip {
		g.WarnConstraints(d)
	}
	return b.String(), nil
}

// typeName checks a name a type is declared under.
func (g *generator) typeName(name string, pos *ir.Position) error {
	switch {
	case !ident.MatchString(name) || strings.HasPrefix(name, "__"):
		return emit.Unsupported(pos, "%s is not a GraphQL name", name)
	case builtin[name]:
		return emit.Unsupported(pos, "%s is a GraphQL built-in scalar", name)
	}
	return nil
}

// object renders an entity, a value, a mixin, or a variant's object type,
// which emit alike. kind says which, and kvs are its own annotations.
func (g *generator) object(b *strings.Builder, name string, meta *ir.Meta, fields []*ir.Field, kind node, kvs []kv) ([]string, error) {
	if len(fields) == 0 {
		return nil, emit.Unsupported(meta.GetPosition(), "%s has no fields, and a GraphQL object needs at least one", name)
	}
	includes := make([]string, len(fields))
	if kind == object {
		includes = unlower.Includes(g.Model, fields)
	}
	for i, inc := range includes {
		if inc != "" && (i == 0 || includes[i-1] != inc) {
			g.Lose(emit.LossInclude, fields[i].GetMeta().GetPosition(), "%s's include of %s is flattened into its fields", name, inc)
		}
	}

	var body strings.Builder
	seen := map[string]bool{}
	for i, f := range fields {
		ref, err := g.Resolve(f.GetType())
		if err != nil {
			return nil, err
		}
		typ, err := g.fieldType(ref)
		if err != nil {
			return nil, err
		}

		field := g.FieldName(f, asWritten)
		if err := claim(seen, name, field, f.GetMeta().GetPosition()); err != nil {
			return nil, err
		}
		doc := emit.Doc(f.GetMeta())
		fkvs := append(g.fieldKVs(name, f, ref, field, includes[i]), g.docKVs(f.GetMeta(), member, "  ", doc)...)
		description(&body, "  ", doc)
		fmt.Fprintf(&body, "  %s: %s%s%s\n", field, typ, deprecated(f.GetMeta()), g.annotations(fkvs))
	}

	doc := typeDoc(meta)
	kvs = append(kvs, g.docKVs(meta, kind, "", doc)...)
	description(b, "", doc)
	fmt.Fprintf(b, "type %s%s {\n%s}\n", name, g.annotations(kvs), body.String())
	return []string{name}, nil
}

// enum renders an enum whose variants carry no fields.
func (g *generator) enum(b *strings.Builder, d *ir.Decl) ([]string, error) {
	name := g.DeclName(d, emit.Pascal)
	if len(d.GetEnumeration().GetVariants()) == 0 {
		return nil, emit.Unsupported(d.GetMeta().GetPosition(), "%s has no variants, and a GraphQL enum needs at least one", name)
	}

	var body strings.Builder
	seen := map[string]bool{}
	for _, v := range d.GetEnumeration().GetVariants() {
		value := g.VariantName(v, emit.ScreamingSnake)
		if value == "true" || value == "false" || value == "null" {
			return nil, emit.Unsupported(v.GetMeta().GetPosition(), "%s.%s is not a GraphQL enum value", name, value)
		}
		if err := claim(seen, name, value, v.GetMeta().GetPosition()); err != nil {
			return nil, err
		}
		var kvs []kv
		if back := emit.FromScreaming(value); back != v.GetMeta().GetName() {
			g.Lose(emit.LossName, v.GetMeta().GetPosition(), "%s.%s is written %s, which reads back as %s", name, v.GetMeta().GetName(), value, back)
			kvs = append(kvs, kv{"name", v.GetMeta().GetName()})
		} else {
			g.CheckRenamed(v.GetDirectives(), emit.ScreamingSnake(back) != value, "%s.%s", name, v.GetMeta().GetName())
		}
		doc := emit.Doc(v.GetMeta())
		kvs = append(kvs, g.docKVs(v.GetMeta(), member, "  ", doc)...)
		description(&body, "  ", doc)
		fmt.Fprintf(&body, "  %s%s%s\n", value, deprecated(v.GetMeta()), g.annotations(kvs))
	}

	doc := typeDoc(d.GetMeta())
	kvs := append(append(g.nameKVs(d, name), g.conformsKV(d)...), g.docKVs(d.GetMeta(), object, "", doc)...)
	description(b, "", doc)
	fmt.Fprintf(b, "enum %s%s {\n%s}\n", name, g.annotations(kvs), body.String())
	return []string{name}, nil
}

// union renders an enum where any variant carries fields: a union of one
// object type per variant.
func (g *generator) union(b *strings.Builder, d *ir.Decl) ([]string, error) {
	name := g.DeclName(d, emit.Pascal)
	declared := []string{name}

	var types strings.Builder
	var members []string
	for _, v := range d.GetEnumeration().GetVariants() {
		member := g.memberName(name, v)
		vname := v.GetMeta().GetName()
		var kvs []kv
		if back := variantBack(name, member); back != vname {
			g.Lose(emit.LossName, v.GetMeta().GetPosition(), "%s.%s is written %s, which reads back as %s", name, vname, member, back)
			kvs = append(kvs, kv{"name", vname})
		} else {
			g.CheckRenamed(v.GetDirectives(), member != name+emit.Pascal(back), "%s.%s", name, vname)
		}
		types.WriteString("\n")

		if len(v.GetFields()) == 0 {
			doc := typeDoc(v.GetMeta())
			if len(doc) > 0 {
				doc = append(doc, "")
			}
			doc = append(doc, placeholderNote(vname))
			kvs = append(kvs, g.docKVs(v.GetMeta(), empty, "", doc)...)
			description(&types, "", doc)
			fmt.Fprintf(&types, "type %s%s {\n  %s: Boolean\n}\n", member, g.annotations(kvs), placeholder)
		} else if _, err := g.object(&types, member, v.GetMeta(), v.GetFields(), object, kvs); err != nil {
			return nil, err
		}
		members = append(members, member)
		declared = append(declared, member)
	}

	doc := typeDoc(d.GetMeta())
	kvs := append(append(g.nameKVs(d, name), g.conformsKV(d)...), g.docKVs(d.GetMeta(), object, "", doc)...)
	description(b, "", doc)
	fmt.Fprintf(b, "union %s%s = %s\n", name, g.annotations(kvs), strings.Join(members, " | "))
	b.WriteString(types.String())
	return declared, nil
}

// variantBack is the variant a union member reads back as: what its name
// adds to the union's, or else its own.
func variantBack(union, member string) string {
	if rest, ok := strings.CutPrefix(member, union); ok && rest != "" {
		return rest
	}
	return member
}

// fieldType returns a field's type: non-null unless the field is optional.
func (g *generator) fieldType(r *emit.Ref) (string, error) {
	r, err := g.Expand(r)
	if err != nil {
		return "", err
	}
	if r.Form != emit.Option && r.Form != emit.Nullable {
		return g.typ(r, true)
	}
	inner, err := g.Expand(r.Elem)
	if err != nil {
		return "", err
	}
	return g.typ(inner, false)
}

// typ is the GraphQL type for a reference, non-null when required.
func (g *generator) typ(r *emit.Ref, required bool) (string, error) {
	r, err := g.Expand(r)
	if err != nil {
		return "", err
	}
	bang := ""
	if required {
		bang = "!"
	}

	switch r.Form {
	case emit.Prim:
		s, ok := scalars[r.Name]
		if !ok {
			return "", emit.Unsupported(r.Pos, "primitive %s has no GraphQL type", r.Name)
		}
		if !builtin[s] {
			if g.local[s] {
				return "", emit.Unsupported(r.Pos, "the scalar %s would collide with the declaration of that name", s)
			}
			g.uses[s] = true
		}
		return s + bang, nil
	case emit.List, emit.Set:
		elem, err := g.Expand(r.Elem)
		if err != nil {
			return "", err
		}
		elemRequired := true
		if elem.Form == emit.Option || elem.Form == emit.Nullable {
			if elem, err = g.Expand(elem.Elem); err != nil {
				return "", err
			}
			elemRequired = false
		}
		e, err := g.typ(elem, elemRequired)
		if err != nil {
			return "", err
		}
		return "[" + e + "]" + bang, nil
	case emit.Map:
		return "", emit.Unsupported(r.Pos, "GraphQL has no map type")
	case emit.Option, emit.Nullable:
		return "", emit.Unsupported(r.Pos, "an optional value holding an optional value has no GraphQL form")
	}
	return g.DeclName(r.Decl, emit.Pascal) + bang, nil
}

func asWritten(name string) string { return name }

// claim takes a member name inside a type.
func claim(seen map[string]bool, owner, name string, pos *ir.Position) error {
	if !ident.MatchString(name) || strings.HasPrefix(name, "__") {
		return emit.Unsupported(pos, "%s.%s is not a GraphQL name", owner, name)
	}
	if seen[name] {
		return emit.Unsupported(pos, "%s has two members named %s in GraphQL", owner, name)
	}
	seen[name] = true
	return nil
}

// typeDoc is a type's description, carrying its deprecation, since a type
// cannot carry @deprecated.
func typeDoc(meta *ir.Meta) []string {
	lines := emit.Doc(meta)
	if reason, ok := emit.Deprecated(meta); ok {
		if len(lines) > 0 {
			lines = append(lines, "")
		}
		if reason == "" {
			lines = append(lines, "Deprecated.")
		} else {
			lines = append(lines, "Deprecated: "+reason)
		}
	}
	return lines
}

// description writes lines as a block string.
func description(b *strings.Builder, indent string, lines []string) {
	if len(lines) == 0 {
		return
	}
	fmt.Fprintf(b, "%s\"\"\"\n", indent)
	for _, line := range lines {
		if line == "" {
			b.WriteString("\n")
			continue
		}
		fmt.Fprintf(b, "%s%s\n", indent, strings.ReplaceAll(line, `"""`, `\"""`))
	}
	fmt.Fprintf(b, "%s\"\"\"\n", indent)
}

// deprecated is a field's or enum value's @deprecated directive, or "".
func deprecated(meta *ir.Meta) string {
	reason, ok := emit.Deprecated(meta)
	switch {
	case !ok:
		return ""
	case reason == "":
		return " @deprecated"
	}
	return " @deprecated(reason: " + literal(reason) + ")"
}
