// Package smithy generates a Smithy IDL 2.0 model from a resolved model,
// and imports one back. The mapping is in docs/architecture/schema-backends.md.
package smithy

import (
	"context"
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/unstoppablemango/tdl/backend/internal/emit"
	"github.com/unstoppablemango/tdl/ir"
	"github.com/unstoppablemango/tdl/plugin"
)

// Name names this backend in a target block and as tdl-gen-smithy.
const Name = "smithy"

// Backend implements [plugin.Backend] and [plugin.Importer].
type Backend struct{}

func (Backend) Describe() plugin.Description {
	str := []ir.LiteralKind{ir.LiteralKind_LITERAL_KIND_STRING}
	return plugin.Description{
		Name:    Name,
		Version: "0.1.0",
		// Each request stands alone.
		Reuse: true,
		Directives: []*plugin.DirectiveSpec{
			// The Smithy namespace; defaults to the model's package.
			{Name: "package", MinArgs: 1, MaxArgs: 1, ArgKinds: str, Placements: plugin.On(plugin.OnBlock)},
			// The Smithy name for a shape, member, enum member, or variant
			// structure.
			{Name: "name", MinArgs: 1, MaxArgs: 1, ArgKinds: str, Placements: plugin.OnMembers()},
			// Write what Smithy cannot say as tdl#source traits and tdl
			// metadata, so import rebuilds the model.
			{Name: "roundtrip", Placements: plugin.On(plugin.OnBlock)},
			{Name: "readback", Placements: plugin.On(plugin.OnBlock)},
		},
		Reverse: true,
	}
}

// preludeShapes maps a TDL primitive to its Smithy prelude shape. Smithy has
// no UUID and no bare date, so those are Strings.
var preludeShapes = map[string]string{
	"string":   "String",
	"int":      "Long",
	"int32":    "Integer",
	"int64":    "Long",
	"uint32":   "Long",
	"uint64":   "BigInteger",
	"float32":  "Float",
	"float64":  "Double",
	"bool":     "Boolean",
	"bytes":    "Blob",
	"decimal":  "BigDecimal",
	"uuid":     "String",
	"instant":  "Timestamp",
	"date":     "String",
	"duration": "String",
}

// simple maps a prelude shape to the keyword declaring a newtype over it.
var simple = map[string]string{
	"String":     "string",
	"Long":       "long",
	"Integer":    "integer",
	"BigInteger": "bigInteger",
	"Float":      "float",
	"Double":     "double",
	"Boolean":    "boolean",
	"Blob":       "blob",
	"BigDecimal": "bigDecimal",
	"Timestamp":  "timestamp",
}

// ident is the Smithy identifier grammar: underscores, then a letter, so
// `_1` and `_` are not identifiers.
var ident = regexp.MustCompile(`^_*[A-Za-z][A-Za-z0-9_]*$`)

type generator struct {
	*emit.Session

	ns string

	// local is every shape name the model's declarations could take, which a
	// same-named prelude reference must qualify around.
	local map[string]bool

	// claimed is every declared shape, keyed by lower case, since Smithy
	// shape IDs are case-insensitive.
	claimed map[string]claim

	// collections is every list and map shape a member needs, by name.
	collections map[string]*shape

	// uses is the collection shapes the declaration being rendered needs.
	uses map[string]bool
}

type claim struct{ name, decl string }

type rendered struct {
	decl   *ir.Decl
	shapes []*shape
	uses   map[string]bool
}

// Generate returns one .smithy file holding the model's declarations, and
// under roundtrip a tdl.smithy defining the trait it uses.
func (Backend) Generate(_ context.Context, req *plugin.Request) (*plugin.Response, error) {
	g := &generator{
		Session:     emit.NewSession(req, "Smithy"),
		local:       map[string]bool{},
		claimed:     map[string]claim{},
		collections: map[string]*shape{},
	}
	g.Roundtrip = g.Bare("roundtrip")
	g.Readback = g.Bare("readback")

	pkg := req.GetModel().GetPackage()
	ns, nsPos := pkg, (*ir.Position)(nil)
	if d, ok := g.Block("package"); ok {
		ns, nsPos = d.GetArgs()[0].GetText(), d.GetPosition()
	}
	// An invalid namespace fails the whole output.
	if !validNamespace(ns) {
		g.Error(nsPos, "%q is not a Smithy namespace", ns)
		return g.Response(nil), nil
	}
	g.ns = ns
	if ns != pkg {
		g.Lose(emit.LossName, nsPos, "package %s is written as namespace %s, which reads back as package %s", pkg, ns, ns)
	}
	if len(req.GetModel().GetDoc()) > 0 {
		g.Lose(emit.LossDoc, nil, "Smithy has no place for the package's doc comment")
	}

	own := g.Own()
	for _, d := range own {
		if d.GetStructure() == nil && d.GetEnumeration() == nil && d.GetNewtype() == nil {
			continue
		}
		name := g.DeclName(d, emit.Pascal)
		g.local[name] = true
		// A union variant carrying fields emits a structure, which is a
		// local name too.
		for _, v := range d.GetEnumeration().GetVariants() {
			if len(v.GetFields()) > 0 {
				g.local[g.variantShape(name, v)] = true
			}
		}
	}

	skipped := map[*ir.Decl]bool{}
	// Each declaration's warnings are held until the cascade decides
	// whether it is emitted, since a skipped one loses everything at once.
	held := map[*ir.Decl][]*plugin.Diagnostic{}
	var out []rendered
	for _, d := range own {
		mark := len(g.Diags)
		g.uses = map[string]bool{}
		shapes, err := g.decl(d)
		if err != nil {
			g.Diags = g.Diags[:mark]
			g.Skip(err)
			skipped[d] = true
			continue
		}
		held[d] = slices.Clone(g.Diags[mark:])
		g.Diags = g.Diags[:mark]
		if len(shapes) > 0 {
			out = append(out, rendered{d, shapes, g.uses})
		}
	}
	g.Cascade(own, skipped)
	for _, d := range own {
		if !skipped[d] {
			g.Diags = append(g.Diags, held[d]...)
		}
	}

	var shapes []*shape
	emitted := map[string]bool{}
	need := map[string]bool{}
	var kept []rendered
	for _, r := range out {
		if skipped[r.decl] {
			continue
		}
		kept = append(kept, r)
		shapes = append(shapes, r.shapes...)
		emitted[r.decl.GetMeta().GetName()] = true
		maps.Copy(need, r.uses)
	}
	for _, name := range slices.Sorted(maps.Keys(need)) {
		shapes = append(shapes, g.collections[name])
	}
	metadata := ""
	if g.Roundtrip {
		g.carry(kept, shapes)
		metadata = g.metadata(emitted, len(shapes) > 0)
	}
	if len(shapes) == 0 && metadata == "" {
		return g.Response(nil), nil
	}

	var b strings.Builder
	b.WriteString("// Code generated by tdl. DO NOT EDIT.\n\n$version: \"2\"\n")
	if metadata != "" {
		b.WriteString("\n")
		b.WriteString(metadata)
	}
	fmt.Fprintf(&b, "\nnamespace %s\n", ns)
	sourced := false
	for _, s := range shapes {
		b.WriteString("\n")
		b.WriteString(g.render(s))
		sourced = sourced || s.traits.source != nil
	}
	files := []*plugin.File{{Path: emit.LastSegment(ns) + ".smithy", Content: []byte(b.String())}}
	if sourced {
		files = append(files, &plugin.File{Path: traitFile, Content: []byte(traitDefinition)})
	}
	return g.Response(files), nil
}

// decl returns the shapes one declaration declares, none for one that
// declares nothing.
func (g *generator) decl(d *ir.Decl) ([]*shape, error) {
	pos := d.GetMeta().GetPosition()
	name := d.GetMeta().GetName()

	if ok, err := g.DeclaresLossy(d, func(id *ir.ID) { g.typeLosses(id, nil) }); !ok {
		return nil, err
	}

	var shapes []*shape
	var err error
	switch e := d.GetEnumeration(); {
	case d.GetNewtype() != nil:
		shapes, err = g.newtype(d)
	case d.GetStructure() != nil:
		var s *shape
		s, err = g.structure(g.DeclName(d, emit.Pascal), d.GetMeta(), d.Fields())
		shapes = []*shape{s}
	case emit.Fielded(e):
		shapes, err = g.union(d)
	default:
		shapes, err = g.enum(d)
	}
	if err != nil {
		return nil, err
	}

	within := map[string]string{}
	for _, s := range shapes {
		n := s.name()
		if !ident.MatchString(n) {
			return nil, emit.Unsupported(pos, "%s would be named %s in Smithy, which is not an identifier", name, n)
		}
		key := fold(n)
		if prior, ok := within[key]; ok {
			return nil, emit.Unsupported(pos, "%s would declare both %s and %s in Smithy, which is one shape twice", name, prior, n)
		}
		within[key] = n
		if other, ok := g.claimed[key]; ok {
			if other.name == n {
				return nil, emit.Unsupported(pos, "%s would declare %s in Smithy, and %s already does", name, n, other.decl)
			}
			return nil, emit.Unsupported(pos, "%s would declare %s in Smithy, and %s declares %s, which Smithy reads as the same shape", name, n, other.decl, other.name)
		}
	}
	// Names are committed only once the declaration renders, so a failed
	// one leaves none behind.
	for _, s := range shapes {
		g.claimed[fold(s.name())] = claim{s.name(), name}
	}
	if len(shapes) > 0 {
		g.losses(d, shapes[0])
	}
	g.WarnConstraints(d)
	return shapes, nil
}

// structure builds an entity, a value, or a mixin, which emit alike.
func (g *generator) structure(name string, meta *ir.Meta, fields []*ir.Field) (*shape, error) {
	s := &shape{id: g.id(name), kind: "structure", traits: metaTraits(meta)}
	seen := map[string]string{}
	for _, f := range fields {
		ref, err := g.Resolve(f.GetType())
		if err != nil {
			return nil, err
		}
		required, target, err := g.memberTarget(ref)
		if err != nil {
			return nil, err
		}

		m := &member{name: g.FieldName(f, asWritten), target: target, traits: metaTraits(f.GetMeta())}
		if err := claimMember(seen, name, m.name, f.GetMeta().GetPosition()); err != nil {
			return nil, err
		}
		m.traits.required = required
		s.members = append(s.members, m)
	}
	return s, nil
}

// enum builds an enum whose variants carry no fields. Each member's value
// is the variant's name as written.
func (g *generator) enum(d *ir.Decl) ([]*shape, error) {
	name := g.DeclName(d, emit.Pascal)
	s := &shape{id: g.id(name), kind: "enum", traits: metaTraits(d.GetMeta())}
	seen := map[string]string{}
	for _, v := range d.GetEnumeration().GetVariants() {
		m := &member{name: g.VariantName(v, emit.ScreamingSnake), target: prelude + "#Unit", traits: metaTraits(v.GetMeta())}
		if err := claimMember(seen, name, m.name, v.GetMeta().GetPosition()); err != nil {
			return nil, err
		}
		value := v.GetMeta().GetName()
		m.traits.enumValue = &value
		s.members = append(s.members, m)
	}
	return []*shape{s}, nil
}

// union builds an enum where any variant carries fields: a union whose
// members target a structure per variant, or Unit for a fieldless one.
func (g *generator) union(d *ir.Decl) ([]*shape, error) {
	name := g.DeclName(d, emit.Pascal)
	u := &shape{id: g.id(name), kind: "union", traits: metaTraits(d.GetMeta())}
	shapes := []*shape{u}

	seen := map[string]string{}
	for _, v := range d.GetEnumeration().GetVariants() {
		m := &member{name: emit.Camel(v.GetMeta().GetName()), target: prelude + "#Unit", traits: metaTraits(v.GetMeta())}
		if err := claimMember(seen, name, m.name, v.GetMeta().GetPosition()); err != nil {
			return nil, err
		}
		if len(v.GetFields()) > 0 {
			s, err := g.structure(g.variantShape(name, v), &ir.Meta{Doc: v.GetMeta().GetDoc()}, v.GetFields())
			if err != nil {
				return nil, err
			}
			m.target = s.id
			shapes = append(shapes, s)
		}
		u.members = append(u.members, m)
	}
	return shapes, nil
}

// newtype builds a newtype over a primitive or a collection as a shape.
// Over a structure or an enum it declares nothing, and a reference targets
// the base.
func (g *generator) newtype(d *ir.Decl) ([]*shape, error) {
	pos := d.GetMeta().GetPosition()
	name := g.DeclName(d, emit.Pascal)

	base, err := g.Resolve(d.GetNewtype().GetBase())
	if err != nil {
		return nil, err
	}
	x, err := g.Expand(base)
	if err != nil {
		return nil, err
	}

	var s *shape
	switch x.Form {
	case emit.Prim:
		prim, ok := preludeShapes[x.Name]
		if !ok {
			return nil, emit.Unsupported(x.Pos, "primitive %s has no Smithy shape", x.Name)
		}
		s = &shape{id: g.id(name), kind: simple[prim]}
	case emit.List, emit.Set, emit.Map:
		if s, err = g.collection(x, name); err != nil {
			return nil, err
		}
	case emit.Option, emit.Nullable:
		return nil, emit.Unsupported(pos, "%s wraps an optional value, and an optional value is not a Smithy shape", d.GetMeta().GetName())
	default:
		g.Lossy(emit.LossNewtype, pos, "newtype %s declares no Smithy shape, and its uses read back as its base", d.GetMeta().GetName())
		return nil, nil
	}
	sparse, unique := s.traits.sparse, s.traits.unique
	s.traits = metaTraits(d.GetMeta())
	s.traits.sparse, s.traits.unique = sparse, unique
	return []*shape{s}, nil
}

// named reports whether a newtype declares a shape.
func (g *generator) named(d *ir.Decl) bool {
	base, err := g.Resolve(d.GetNewtype().GetBase())
	if err != nil {
		return true
	}
	x, err := g.Expand(base)
	if err != nil {
		return true
	}
	return x.Form != emit.Named
}

// memberTarget returns whether a member is required, and its target.
func (g *generator) memberTarget(r *emit.Ref) (bool, string, error) {
	if r.Form != emit.Option && r.Form != emit.Nullable {
		target, err := g.target(r)
		return true, target, err
	}
	if r.Elem.Form == emit.Option || r.Elem.Form == emit.Nullable {
		return false, "", emit.Unsupported(r.Pos, "an optional value holding an optional value has no Smithy form")
	}
	target, err := g.target(r.Elem)
	return false, target, err
}

// target is the ID of the shape a reference targets.
func (g *generator) target(r *emit.Ref) (string, error) {
	switch r.Form {
	case emit.Prim:
		prim, ok := preludeShapes[r.Name]
		if !ok {
			return "", emit.Unsupported(r.Pos, "primitive %s has no Smithy shape", r.Name)
		}
		return prelude + "#" + prim, nil
	case emit.List, emit.Set, emit.Map:
		s, err := g.collection(r, "")
		if err != nil {
			return "", err
		}
		name := s.name()
		if g.local[name] {
			return "", emit.Unsupported(r.Pos, "the shape %s this collection needs would collide with the declaration of that name", name)
		}
		// Two different collections can derive the same name.
		if prior, ok := g.collections[name]; ok && g.render(prior) != g.render(s) {
			return "", emit.Unsupported(r.Pos, "the shape %s this collection needs is already declared holding something else", name)
		}
		g.collections[name] = s
		g.uses[name] = true
		return s.id, nil
	case emit.Option, emit.Nullable:
		return "", emit.Unsupported(r.Pos, "an optional value here has no Smithy form")
	}

	if r.Decl.GetNewtype() != nil && !g.named(r.Decl) {
		next, err := g.Resolve(r.Decl.GetNewtype().GetBase())
		if err != nil {
			return "", err
		}
		return g.target(next)
	}
	return g.id(g.DeclName(r.Decl, emit.Pascal)), nil
}

// collection builds a list or map shape. A newtype passes its name; ""
// names the shape for what it holds. An optional element makes the shape
// @sparse.
func (g *generator) collection(r *emit.Ref, name string) (*shape, error) {
	elem, sparse := r.Elem, false
	if elem.Form == emit.Option || elem.Form == emit.Nullable {
		elem, sparse = elem.Elem, true
		if elem.Form == emit.Option || elem.Form == emit.Nullable {
			return nil, emit.Unsupported(r.Pos, "an optional value holding an optional value has no Smithy form")
		}
	}
	if sparse && r.Form == emit.Set {
		return nil, emit.Unsupported(r.Pos, "a set holding optional values has no Smithy form, since a @uniqueItems list cannot be @sparse")
	}
	value, err := g.target(elem)
	if err != nil {
		return nil, err
	}

	if r.Form == emit.Map {
		kx, err := g.Expand(r.Key)
		if err != nil {
			return nil, err
		}
		stringy := kx.Form == emit.Prim && preludeShapes[kx.Name] == "String"
		enum := kx.Form == emit.Named && kx.Decl.GetEnumeration() != nil && !emit.Fielded(kx.Decl.GetEnumeration())
		if !stringy && !enum {
			return nil, emit.Unsupported(r.Pos, "a Smithy map key is a string or an enum")
		}
		key, err := g.target(r.Key)
		if err != nil {
			return nil, err
		}
		if name == "" {
			name = mapName(key, value, sparse)
		}
		return &shape{id: g.id(name), kind: "map", traits: traits{sparse: sparse},
			key: &member{name: "key", target: key}, value: &member{name: "value", target: value}}, nil
	}

	if name == "" {
		name = listName(value, sparse, r.Form == emit.Set)
	}
	return &shape{id: g.id(name), kind: "list", traits: traits{sparse: sparse, unique: r.Form == emit.Set},
		member: &member{name: "member", target: value}}, nil
}

// listName is the name a list shape takes for what it holds.
func listName(elem string, sparse, unique bool) string {
	suffix := "List"
	if unique {
		suffix = "Set"
	}
	return sparsePrefix(sparse) + bare(elem) + suffix
}

// mapName is the name a map shape takes for what it holds.
func mapName(key, value string, sparse bool) string {
	return sparsePrefix(sparse) + bare(key) + bare(value) + "Map"
}

func (g *generator) id(name string) string { return g.ns + "#" + name }

// ref spells a shape ID inside the generated namespace: a local shape and
// a prelude shape by name, the prelude one qualified when a model shape
// takes its name.
func (g *generator) ref(id string) string {
	ns, name, _ := strings.Cut(id, "#")
	if ns == prelude && g.local[name] || ns != prelude && ns != g.ns {
		return id
	}
	return name
}

// bare is a shape ID's name.
func bare(id string) string {
	_, name, _ := strings.Cut(id, "#")
	return name
}

func sparsePrefix(sparse bool) string {
	if sparse {
		return "Sparse"
	}
	return ""
}

func asWritten(name string) string { return name }

// claimMember takes a member name inside a shape, case-insensitively, as
// Smithy compares member names.
func claimMember(seen map[string]string, owner, name string, pos *ir.Position) error {
	if !ident.MatchString(name) {
		return emit.Unsupported(pos, "%s.%s is not a Smithy identifier", owner, name)
	}
	if prior, ok := seen[fold(name)]; ok {
		if prior == name {
			return emit.Unsupported(pos, "%s has two members named %s in Smithy", owner, name)
		}
		return emit.Unsupported(pos, "%s has members named %s and %s, which Smithy reads as one member", owner, prior, name)
	}
	seen[fold(name)] = name
	return nil
}

func fold(name string) string { return strings.ToLower(name) }

// variantShape names the structure a fielded variant of union name emits.
func (g *generator) variantShape(name string, v *ir.Variant) string {
	return g.VariantName(v, func(s string) string { return name + emit.Pascal(s) })
}

// metaTraits is a node's documentation and deprecation as traits.
func metaTraits(meta *ir.Meta) traits {
	var t traits
	if lines := emit.Doc(meta); len(lines) > 0 {
		doc := strings.Join(lines, "\n")
		t.doc = &doc
	}
	if reason, ok := emit.Deprecated(meta); ok {
		t.deprecated = &deprecation{reason}
	}
	return t
}

// render writes a shape as Smithy IDL.
func (g *generator) render(s *shape) string {
	var b strings.Builder
	shapeTraits(&b, "", s.traits)
	switch s.kind {
	case "structure", "union":
		fmt.Fprintf(&b, "%s %s {\n", s.kind, s.name())
		for _, m := range s.members {
			comment(&b, "    ", m.traits.doc)
			if m.traits.required {
				b.WriteString("    @required\n")
			}
			deprecated(&b, "    ", m.traits.deprecated)
			fmt.Fprintf(&b, "    %s: %s\n", m.name, g.ref(m.target))
		}
		b.WriteString("}\n")
	case "enum":
		fmt.Fprintf(&b, "enum %s {\n", s.name())
		for _, m := range s.members {
			comment(&b, "    ", m.traits.doc)
			deprecated(&b, "    ", m.traits.deprecated)
			fmt.Fprintf(&b, "    %s = %s\n", m.name, strconv.Quote(*m.traits.enumValue))
		}
		b.WriteString("}\n")
	case "list":
		fmt.Fprintf(&b, "list %s {\n    member: %s\n}\n", s.name(), g.ref(s.member.target))
	case "map":
		fmt.Fprintf(&b, "map %s {\n    key: %s\n    value: %s\n}\n", s.name(), g.ref(s.key.target), g.ref(s.value.target))
	default:
		fmt.Fprintf(&b, "%s %s\n", s.kind, s.name())
	}
	return b.String()
}

// shapeTraits writes a shape's traits, its doc comment first.
func shapeTraits(b *strings.Builder, indent string, t traits) {
	comment(b, indent, t.doc)
	deprecated(b, indent, t.deprecated)
	if t.source != nil {
		fmt.Fprintf(b, "%s@%s(%s)\n", indent, traitSource, quote(*t.source))
	}
	if t.sparse {
		fmt.Fprintf(b, "%s@sparse\n", indent)
	}
	if t.unique {
		fmt.Fprintf(b, "%s@uniqueItems\n", indent)
	}
}

// comment writes documentation as Smithy doc comments.
func comment(b *strings.Builder, indent string, doc *string) {
	if doc == nil {
		return
	}
	for line := range strings.SplitSeq(*doc, "\n") {
		if line == "" {
			fmt.Fprintf(b, "%s///\n", indent)
			continue
		}
		fmt.Fprintf(b, "%s/// %s\n", indent, line)
	}
}

// deprecated writes the @deprecated trait, with any reason as its message.
func deprecated(b *strings.Builder, indent string, d *deprecation) {
	switch {
	case d == nil:
	case d.message == "":
		fmt.Fprintf(b, "%s@deprecated\n", indent)
	default:
		fmt.Fprintf(b, "%s@deprecated(message: %s)\n", indent, quote(d.message))
	}
}

func validNamespace(ns string) bool {
	if ns == "" {
		return false
	}
	for seg := range strings.SplitSeq(ns, ".") {
		if !ident.MatchString(seg) {
			return false
		}
	}
	return true
}
