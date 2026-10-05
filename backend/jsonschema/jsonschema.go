// Package jsonschema generates a JSON Schema document from a resolved
// model.
//
// docs/design/schema-backends.md has the mapping and the reasons for it.
// Every declaration is a definition under $defs, and an enum whose
// variants carry fields is a oneOf discriminated on a `kind` property, as
// in the TypeScript backend. JSON Schema is the one schema target that can
// state TDL's constraints, so a `where` block becomes the keywords that
// check it rather than a warning.
package jsonschema

import (
	"bytes"
	"context"
	"regexp"
	"strings"

	"github.com/unstoppablemango/tdl/backend/internal/emit"
	"github.com/unstoppablemango/tdl/ir"
	"github.com/unstoppablemango/tdl/plugin"
)

// Name is what this backend is called, in a target block and as
// tdl-gen-jsonschema on PATH.
const Name = "jsonschema"

// Backend implements [plugin.Backend].
type Backend struct{}

func (Backend) Describe() plugin.Description {
	str := []ir.LiteralKind{ir.LiteralKind_LITERAL_KIND_STRING}
	return plugin.Description{
		Name:    Name,
		Version: "0.1.0",
		// Each request is answered from the request alone.
		Reuse: true,
		Directives: []*plugin.DirectiveSpec{
			// The definition's name for a type, or a field's property name.
			{Name: "name", MinArgs: 1, MaxArgs: 1, ArgKinds: str},
			// The property a fielded enum's variants are told apart by.
			// Written on the target block it is the default, and on an enum
			// it wins.
			{Name: "discriminant", MinArgs: 1, MaxArgs: 1, ArgKinds: str},
			// The dialect: "2020-12", the default, or "draft-07".
			{Name: "draft", MinArgs: 1, MaxArgs: 1, ArgKinds: str},
			// The declaration the document itself validates.
			{Name: "root", MinArgs: 1, MaxArgs: 1, ArgKinds: str},
			// The document's $id.
			{Name: "id", MinArgs: 1, MaxArgs: 1, ArgKinds: str},
			// Refuses properties a struct or variant does not declare. On the
			// target block it applies to every one.
			{Name: "closed"},
		},
	}
}

// dialect is what differs between the drafts this backend writes.
type dialect struct {
	schema string
	defs   string
	// siblings is whether keywords beside a $ref apply. Draft 7 ignores
	// them, so a $ref with siblings is moved into an allOf.
	siblings bool
	// deprecated is whether the `deprecated` keyword exists.
	deprecated bool
}

var dialects = map[string]dialect{
	"2020-12":  {schema: "https://json-schema.org/draft/2020-12/schema", defs: "$defs", siblings: true, deprecated: true},
	"draft-07": {schema: "http://json-schema.org/draft-07/schema#", defs: "definitions"},
}

const defaultDialect = "2020-12"

// scalar is the JSON form of a TDL primitive.
type scalar struct {
	typ    string
	format string
	// encoding is the contentEncoding of a string holding binary data.
	encoding string
	min, max int64
	bounded  bool
	unsigned bool
}

// scalars maps a TDL primitive to the JSON value it is. JSON has no bytes,
// decimal, UUID, or time, so each is a string, with the format that says
// which where JSON Schema has one. A decimal is a string rather than a
// number for the reason the Go backend gives: a number loses its exactness
// in most parsers.
var scalars = map[string]scalar{
	"string":   {typ: "string"},
	"int":      {typ: "integer"},
	"int32":    {typ: "integer", min: -1 << 31, max: 1<<31 - 1, bounded: true},
	"int64":    {typ: "integer"},
	"uint32":   {typ: "integer", min: 0, max: 1<<32 - 1, bounded: true},
	"uint64":   {typ: "integer", unsigned: true},
	"float32":  {typ: "number"},
	"float64":  {typ: "number"},
	"bool":     {typ: "boolean"},
	"bytes":    {typ: "string", encoding: "base64"},
	"decimal":  {typ: "string"},
	"uuid":     {typ: "string", format: "uuid"},
	"instant":  {typ: "string", format: "date-time"},
	"date":     {typ: "string", format: "date"},
	"duration": {typ: "string", format: "duration"},
}

// defName is what a definition may be called: it is written into a JSON
// pointer inside a URI fragment, where anything else needs escaping.
var defName = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_.-]*$`)

// defaultDiscriminant is the property a fielded enum's variants carry when
// no directive names one.
const defaultDiscriminant = "kind"

type generator struct {
	*emit.Session
	dialect dialect

	// discriminant is the target block's default.
	discriminant string
	// closed is whether the target block closes every object.
	closed bool

	// names is every definition name, with the declaration that took it.
	names map[string]string
}

// Generate returns one .schema.json file holding every declaration the
// model owns.
func (Backend) Generate(_ context.Context, req *plugin.Request) (*plugin.Response, error) {
	g := &generator{
		Session:      emit.NewSession(req, "JSON Schema"),
		dialect:      dialects[defaultDialect],
		discriminant: defaultDiscriminant,
		names:        map[string]string{},
	}
	if d, ok := g.Block("draft"); ok {
		v, known := dialects[d.GetArgs()[0].GetText()]
		if !known {
			g.Error(d.GetPosition(), "%q is not a JSON Schema draft this backend writes; use \"2020-12\" or \"draft-07\"", d.GetArgs()[0].GetText())
			return g.Response(nil), nil
		}
		g.dialect = v
	}
	if d, ok := g.Block("discriminant"); ok {
		g.discriminant = d.GetArgs()[0].GetText()
	}
	g.closed = g.blockTagged("closed")

	own := g.Own()
	defs := map[*ir.Decl]*object{}
	skipped := map[*ir.Decl]bool{}
	for _, d := range own {
		mark := len(g.Diags)
		s, err := g.decl(d)
		if err != nil {
			// A skipped declaration's constraint warnings describe output
			// that is not written.
			g.Diags = g.Diags[:mark]
			g.Warn(err)
			skipped[d] = true
			continue
		}
		defs[d] = s
	}
	g.Cascade(own, skipped)

	all := newObject()
	for _, d := range own {
		if s := defs[d]; s != nil && !skipped[d] {
			all.set(g.DeclName(d, emit.Pascal), s)
		}
	}
	if len(all.keys) == 0 {
		return g.Response(nil), nil
	}

	doc := newObject().set("$schema", g.dialect.schema)
	if d, ok := g.Block("id"); ok {
		doc.set("$id", d.GetArgs()[0].GetText())
	}
	doc.set("$comment", "Code generated by tdl. DO NOT EDIT.")
	if d, ok := g.Block("root"); ok {
		name := d.GetArgs()[0].GetText()
		decl := g.find(own, name)
		switch {
		case decl == nil:
			g.Error(d.GetPosition(), "root names %s, which the model does not declare", name)
			return g.Response(nil), nil
		case defs[decl] == nil || skipped[decl]:
			g.Error(d.GetPosition(), "root names %s, which is not generated", name)
			return g.Response(nil), nil
		}
		doc.set("$ref", g.ref(decl))
	}
	doc.set(g.dialect.defs, all)

	if !g.dialect.siblings {
		hoist(doc)
	}
	var b bytes.Buffer
	write(&b, doc, "")
	b.WriteString("\n")

	path := "model.schema.json"
	if pkg := req.GetModel().GetPackage(); pkg != "" {
		path = emit.LastSegment(pkg) + ".schema.json"
	}
	return g.Response([]*plugin.File{{Path: path, Content: b.Bytes()}}), nil
}

// find is the declaration among decls a TDL name names.
func (g *generator) find(decls []*ir.Decl, name string) *ir.Decl {
	for _, d := range decls {
		if d.GetMeta().GetName() == name || emit.LastSegment(d.GetMeta().GetName()) == name {
			return d
		}
	}
	return nil
}

// decl renders one declaration's definition, or nil for one that declares
// nothing.
func (g *generator) decl(d *ir.Decl) (*object, error) {
	pos := d.GetMeta().GetPosition()
	name := d.GetMeta().GetName()

	switch {
	case d.GetClass() != nil:
		return nil, emit.Unsupported(pos, "%s is a class, and classes are not generated yet", name)
	case d.GetUnit() != nil:
		return nil, emit.Unsupported(pos, "%s is a unit, and units are not generated yet", name)
	case d.GetStructure() == nil && d.GetEnumeration() == nil && d.GetNewtype() == nil:
		// An alias is expanded where it is used, and a model's own
		// primitive names an opaque root; neither declares anything.
		return nil, nil
	}
	if len(d.Params()) > 0 {
		return nil, emit.Unsupported(pos, "%s is parameterized, and generics are not generated yet", name)
	}

	def := g.DeclName(d, emit.Pascal)
	if !defName.MatchString(def) {
		return nil, emit.Unsupported(pos, "%s would be defined as %q, which is not a JSON Schema definition name", name, def)
	}
	if other, ok := g.names[def]; ok {
		return nil, emit.Unsupported(pos, "%s would be defined as %s, and %s already is", name, def, other)
	}

	s := newObject()
	g.describe(s, d.GetMeta())
	var err error
	switch e := d.GetEnumeration(); {
	case d.GetNewtype() != nil:
		err = g.newtype(s, d)
	case d.GetStructure() != nil:
		err = g.object(s, def, "", "", d.Fields(), g.closed || g.tagged(d.GetDirectives(), "closed"))
	case emit.Fielded(e):
		err = g.union(s, d)
	default:
		values := make([]any, len(e.GetVariants()))
		for i, v := range e.GetVariants() {
			values[i] = v.GetMeta().GetName()
		}
		s.set("type", "string").set("enum", values)
	}
	if err != nil {
		return nil, err
	}
	g.names[def] = name
	return s, nil
}

// newtype renders a newtype as its base with the constraints written on
// it. A base that is itself a newtype is a $ref, and the constraints it
// carries are checked there, so only this one's own are written.
func (g *generator) newtype(s *object, d *ir.Decl) error {
	ref, err := g.Resolve(d.GetNewtype().GetBase())
	if err != nil {
		return err
	}
	base, err := g.typ(ref)
	if err != nil {
		return err
	}
	for _, k := range base.keys {
		s.set(k, base.vals[k])
	}

	var own []*ir.Constraint
	for _, c := range d.GetNewtype().GetValueConstraints() {
		if c.GetFrom() == nil {
			own = append(own, c)
		}
	}
	g.constrain(s, g.kindOf(ref), emit.LastSegment(d.GetMeta().GetName()), own)
	return nil
}

// object renders a struct or a variant. An entity, a value, and a mixin
// differ in what they mean and not in what they emit. A variant passes
// the discriminant and its value, which come first.
func (g *generator) object(s *object, owner, disc, tag string, fields []*ir.Field, closed bool) error {
	s.set("type", "object")
	props := newObject()
	var required []any
	if disc != "" {
		props.set(disc, newObject().set("const", tag))
		required = append(required, disc)
	}

	for _, f := range fields {
		prop := g.FieldName(f, asWritten)
		if _, dup := props.get(prop); dup {
			if prop == disc {
				return emit.Unsupported(f.GetMeta().GetPosition(),
					"%s.%s takes the name of the discriminant, and a discriminant directive can rename it", owner, prop)
			}
			return emit.Unsupported(f.GetMeta().GetPosition(), "%s has two properties named %s", owner, prop)
		}
		schema, optional, err := g.property(owner, f)
		if err != nil {
			return err
		}
		props.set(prop, schema)
		if !optional {
			required = append(required, prop)
		}
	}

	if len(props.keys) > 0 {
		s.set("properties", props)
	}
	if len(required) > 0 {
		s.set("required", required)
	}
	if closed {
		s.set("additionalProperties", false)
	}
	return nil
}

// union renders an enum where any variant carries fields: a oneOf of one
// object per variant, each holding the discriminant with the variant's name
// as its value.
func (g *generator) union(s *object, d *ir.Decl) error {
	owner := emit.LastSegment(d.GetMeta().GetName())
	disc := g.discriminant
	if t, ok := g.Text(d.GetDirectives(), "discriminant"); ok {
		disc = t
	}
	closed := g.closed || g.tagged(d.GetDirectives(), "closed")

	var members []any
	for _, v := range d.GetEnumeration().GetVariants() {
		m := newObject().set("title", v.GetMeta().GetName())
		g.describe(m, v.GetMeta())
		if err := g.object(m, owner+"."+v.GetMeta().GetName(), disc, v.GetMeta().GetName(), v.GetFields(), closed); err != nil {
			return err
		}
		members = append(members, m)
	}
	s.set("oneOf", members)
	return nil
}

// property renders a field's schema and reports whether its key may be
// absent. A `T | null` field is required and may hold null; a `T?` one may
// be left out.
func (g *generator) property(owner string, f *ir.Field) (*object, bool, error) {
	r, err := g.Resolve(f.GetType())
	if err != nil {
		return nil, false, err
	}
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
		return nil, false, err
	}
	if len(f.GetConstraints()) > 0 {
		g.constrain(s, g.kindOf(r), owner+"."+f.GetMeta().GetName(), f.GetConstraints())
	}
	if nullable {
		s = orNull(s)
	}

	out := newObject()
	g.describe(out, f.GetMeta())
	for _, k := range s.keys {
		out.set(k, s.vals[k])
	}
	return out, optional, nil
}

// typ is the schema for a reference. An optional value inside a collection
// is one that may be null, since JSON has no absent element.
func (g *generator) typ(r *emit.Ref) (*object, error) {
	switch r.Form {
	case emit.Prim:
		sc, ok := scalars[r.Name]
		if !ok {
			return nil, emit.Unsupported(r.Pos, "primitive %s has no JSON Schema type", r.Name)
		}
		s := newObject().set("type", sc.typ)
		if sc.format != "" {
			s.set("format", sc.format)
		}
		if sc.encoding != "" {
			s.set("contentEncoding", sc.encoding)
		}
		if sc.bounded {
			s.set("minimum", sc.min).set("maximum", sc.max)
		}
		if sc.unsigned {
			s.set("minimum", int64(0))
		}
		return s, nil
	case emit.List, emit.Set:
		elem, err := g.typ(r.Elem)
		if err != nil {
			return nil, err
		}
		s := newObject().set("type", "array").set("items", elem)
		if r.Form == emit.Set {
			s.set("uniqueItems", true)
		}
		return s, nil
	case emit.Map:
		return g.dict(r)
	case emit.Option, emit.Nullable:
		inner, err := g.typ(r.Elem)
		if err != nil {
			return nil, err
		}
		return orNull(inner), nil
	}
	return newObject().set("$ref", g.ref(r.Decl)), nil
}

// dict renders a map as an object whose values are the map's. A JSON
// object's keys are strings, so a key is anything that is a string in
// JSON, an integer, written as its digits, or a fieldless enum.
func (g *generator) dict(r *emit.Ref) (*object, error) {
	val, err := g.typ(r.Elem)
	if err != nil {
		return nil, err
	}
	s := newObject().set("type", "object")

	kx, err := g.Expand(r.Key)
	if err != nil {
		return nil, err
	}
	switch {
	case kx.Form == emit.Prim && scalars[kx.Name].typ == "string":
		if r.Key.Form == emit.Named {
			s.set("propertyNames", newObject().set("$ref", g.ref(r.Key.Decl)))
		}
	case kx.Form == emit.Prim && scalars[kx.Name].typ == "integer":
		s.set("propertyNames", newObject().set("pattern", `^-?(0|[1-9][0-9]*)$`))
	case kx.Form == emit.Named && kx.Decl.GetEnumeration() != nil && !emit.Fielded(kx.Decl.GetEnumeration()):
		s.set("propertyNames", newObject().set("$ref", g.ref(kx.Decl)))
	default:
		return nil, emit.Unsupported(r.Pos, "a JSON object key is a string, an integer, or a fieldless enum, and this one is not")
	}
	s.set("additionalProperties", val)
	return s, nil
}

// orNull is a schema that also accepts null.
func orNull(s *object) *object {
	if anyOf, ok := s.get("anyOf"); ok && len(s.keys) == 1 {
		for _, m := range anyOf.([]any) {
			if t, _ := m.(*object).get("type"); t == "null" {
				return s
			}
		}
	}
	return newObject().set("anyOf", []any{s, newObject().set("type", "null")})
}

// ref is the JSON pointer to a declaration's definition.
func (g *generator) ref(d *ir.Decl) string {
	return "#/" + g.dialect.defs + "/" + g.DeclName(d, emit.Pascal)
}

// describe writes a node's documentation, and its deprecation, which draft
// 7 has no keyword for and so says in the description.
func (g *generator) describe(s *object, meta *ir.Meta) {
	lines := emit.Doc(meta)
	reason, deprecated := emit.Deprecated(meta)
	if deprecated && !g.dialect.deprecated {
		if len(lines) > 0 {
			lines = append(lines, "")
		}
		lines = append(lines, strings.TrimSpace("Deprecated. "+reason))
	}
	if len(lines) > 0 {
		s.set("description", strings.Join(lines, "\n"))
	}
	if deprecated && g.dialect.deprecated {
		s.set("deprecated", true)
	}
}

// tagged reports whether a node carries an argument-less directive.
func (g *generator) tagged(all []*ir.Directive, name string) bool {
	for _, d := range plugin.Directives(g.Target, all) {
		if d.GetName() == name {
			return true
		}
	}
	return false
}

// blockTagged reports whether the target block carries an argument-less
// directive.
func (g *generator) blockTagged(name string) bool {
	for _, block := range g.Model.GetTargets() {
		if block.GetMeta().GetName() == g.Target && g.tagged(block.GetDirectives(), name) {
			return true
		}
	}
	return false
}

// hoist moves every $ref with siblings into an allOf, for a dialect that
// ignores a $ref's siblings.
func hoist(v any) {
	switch v := v.(type) {
	case *object:
		for _, k := range v.keys {
			hoist(v.vals[k])
		}
		ref, ok := v.get("$ref")
		if !ok || len(v.keys) == 1 {
			return
		}
		v.del("$ref")
		v.set("allOf", []any{newObject().set("$ref", ref)})
	case []any:
		for _, e := range v {
			hoist(e)
		}
	}
}

func asWritten(name string) string { return name }
