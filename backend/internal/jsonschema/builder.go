// Package jsonschema builds JSON Schema definitions from a resolved model.
//
// It serves every backend whose output is JSON Schema or a dialect of it:
// the jsonschema backend and the openapi backend. A [Dialect] states what
// differs between them, and the backend writes the document around the
// definitions.
package jsonschema

import (
	"regexp"
	"strings"

	"github.com/unstoppablemango/tdl/backend/internal/emit"
	"github.com/unstoppablemango/tdl/ir"
	"github.com/unstoppablemango/tdl/plugin"
)

// Null is how a dialect says a value may also be null.
type Null int

const (
	// NullType is an anyOf with {"type": "null"}.
	NullType Null = iota
	// NullNullable is OpenAPI 3.0's `nullable` keyword.
	NullNullable
	// NullExtension is the `x-nullable` extension, OpenAPI 2.0's
	// convention, since 2.0 has no way to say it.
	NullExtension
)

// Unions is how a dialect writes an enum whose variants carry fields.
type Unions int

const (
	// UnionsInline is a oneOf of one inline object per variant.
	UnionsInline Unions = iota
	// UnionsReferenced is a oneOf of a $ref per variant, each variant its
	// own definition, with an OpenAPI discriminator object naming them.
	UnionsReferenced
	// UnionsNone is a dialect with no oneOf, where such an enum is a
	// warning.
	UnionsNone
)

// Dialect is what differs between the documents built here.
type Dialect struct {
	// Name is the dialect as a message names it.
	Name string
	// Ref is the JSON pointer prefix a definition is found under, such as
	// "#/$defs/".
	Ref string
	// Siblings is whether keywords beside a $ref apply. Where they do not,
	// [Hoist] moves a $ref with siblings into an allOf.
	Siblings bool
	// Deprecated is whether the `deprecated` keyword exists. Where it does
	// not, the description says so.
	Deprecated bool
	// Const is whether the `const` keyword exists. Where it does not, a
	// one-member enum stands in.
	Const bool
	Null  Null
	// ContentEncoding is whether bytes are written with contentEncoding.
	// Where they are not, they are OpenAPI's `format: byte`.
	ContentEncoding bool
	// PropertyNames is whether a map key's schema can be stated.
	PropertyNames bool
	Unions        Unions
	// Formats is whether a number carries OpenAPI's width formats: int32,
	// int64, float, and double.
	Formats bool
}

// scalar is the JSON form of a TDL primitive.
type scalar struct {
	typ    string
	format string
	// width is the OpenAPI format naming the number's width.
	width string
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
	"int":      {typ: "integer", width: "int64"},
	"int32":    {typ: "integer", width: "int32", min: -1 << 31, max: 1<<31 - 1, bounded: true},
	"int64":    {typ: "integer", width: "int64"},
	"uint32":   {typ: "integer", width: "int64", min: 0, max: 1<<32 - 1, bounded: true},
	"uint64":   {typ: "integer", unsigned: true},
	"float32":  {typ: "number", width: "float"},
	"float64":  {typ: "number", width: "double"},
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

// DefaultDiscriminant is the property a fielded enum's variants carry when
// no directive names one.
const DefaultDiscriminant = "kind"

// Builder turns declarations into definitions in one dialect.
type Builder struct {
	*emit.Session
	Dialect Dialect

	// Discriminant is the target block's default.
	Discriminant string
	// Closed is whether the target block closes every object.
	Closed bool

	// names is every definition name, with the declaration that took it.
	names map[string]string
}

// New returns a builder reading the target block's `discriminant` and
// `closed` directives.
func New(s *emit.Session, d Dialect) *Builder {
	b := &Builder{Session: s, Dialect: d, Discriminant: DefaultDiscriminant, names: map[string]string{}}
	if d, ok := b.Block("discriminant"); ok {
		b.Discriminant = d.GetArgs()[0].GetText()
	}
	b.Closed = b.BlockTagged("closed")
	return b
}

type def struct {
	name   string
	schema *Object
}

// Build returns a definition for every declaration in own that declares
// one, keyed by name in declaration order, and the declarations it built.
// One that cannot be built is a warning, and so is every declaration
// naming it.
func (b *Builder) Build(own []*ir.Decl) (*Object, map[*ir.Decl]bool) {
	defs := map[*ir.Decl][]def{}
	skipped := map[*ir.Decl]bool{}
	for _, d := range own {
		mark := len(b.Diags)
		ds, err := b.decl(d)
		if err != nil {
			// A skipped declaration's constraint warnings describe output
			// that is not written.
			b.Diags = b.Diags[:mark]
			b.Warn(err)
			skipped[d] = true
			continue
		}
		defs[d] = ds
	}
	b.Cascade(own, skipped)

	all := NewObject()
	built := map[*ir.Decl]bool{}
	for _, d := range own {
		if skipped[d] || len(defs[d]) == 0 {
			continue
		}
		built[d] = true
		for _, x := range defs[d] {
			all.Set(x.name, x.schema)
		}
	}
	return all, built
}

// decl renders one declaration's definitions, or none for one that
// declares nothing.
func (b *Builder) decl(d *ir.Decl) ([]def, error) {
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

	n := b.DeclName(d, emit.Pascal)
	if err := b.free(pos, name, n); err != nil {
		return nil, err
	}

	s := NewObject()
	b.describe(s, d.GetMeta())
	defs := []def{{n, s}}
	var err error
	switch e := d.GetEnumeration(); {
	case d.GetNewtype() != nil:
		err = b.newtype(s, d)
	case d.GetStructure() != nil:
		err = b.object(s, n, "", "", d.Fields(), b.Closed || b.Tagged(d.GetDirectives(), "closed"))
	case emit.Fielded(e):
		var variants []def
		variants, err = b.union(s, d, n)
		defs = append(defs, variants...)
	default:
		values := make([]any, len(e.GetVariants()))
		for i, v := range e.GetVariants() {
			values[i] = v.GetMeta().GetName()
		}
		s.Set("type", "string").Set("enum", values)
	}
	if err != nil {
		return nil, err
	}
	for _, x := range defs {
		b.names[x.name] = name
	}
	return defs, nil
}

// free reports an error unless a definition may take name.
func (b *Builder) free(pos *ir.Position, owner, name string) error {
	if !defName.MatchString(name) {
		return emit.Unsupported(pos, "%s would be defined as %q, which is not a %s definition name", owner, name, b.Dialect.Name)
	}
	if other, ok := b.names[name]; ok {
		return emit.Unsupported(pos, "%s would be defined as %s, and %s already is", owner, name, other)
	}
	return nil
}

// newtype renders a newtype as its base with the constraints written on
// it. A base that is itself a newtype is a $ref, and the constraints it
// carries are checked there, so only this one's own are written.
func (b *Builder) newtype(s *Object, d *ir.Decl) error {
	ref, err := b.Resolve(d.GetNewtype().GetBase())
	if err != nil {
		return err
	}
	base, err := b.typ(ref)
	if err != nil {
		return err
	}
	for _, k := range base.keys {
		s.Set(k, base.vals[k])
	}

	var own []*ir.Constraint
	for _, c := range d.GetNewtype().GetValueConstraints() {
		if c.GetFrom() == nil {
			own = append(own, c)
		}
	}
	b.constrain(s, b.kindOf(ref), emit.LastSegment(d.GetMeta().GetName()), own)
	return nil
}

// object renders a struct or a variant. An entity, a value, and a mixin
// differ in what they mean and not in what they emit. A variant passes
// the discriminant and its value, which come first.
func (b *Builder) object(s *Object, owner, disc, tag string, fields []*ir.Field, closed bool) error {
	s.Set("type", "object")
	props := NewObject()
	var required []any
	if disc != "" {
		props.Set(disc, b.constant(tag))
		required = append(required, disc)
	}

	for _, f := range fields {
		prop := b.FieldName(f, asWritten)
		if _, dup := props.Get(prop); dup {
			if prop == disc {
				return emit.Unsupported(f.GetMeta().GetPosition(),
					"%s.%s takes the name of the discriminant, and a discriminant directive can rename it", owner, prop)
			}
			return emit.Unsupported(f.GetMeta().GetPosition(), "%s has two properties named %s", owner, prop)
		}
		schema, optional, err := b.property(owner, f)
		if err != nil {
			return err
		}
		props.Set(prop, schema)
		if !optional {
			required = append(required, prop)
		}
	}

	if len(props.keys) > 0 {
		s.Set("properties", props)
	}
	if len(required) > 0 {
		s.Set("required", required)
	}
	if closed {
		s.Set("additionalProperties", false)
	}
	return nil
}

// constant is a schema accepting one string.
func (b *Builder) constant(v string) *Object {
	if b.Dialect.Const {
		return NewObject().Set("const", v)
	}
	return NewObject().Set("type", "string").Set("enum", []any{v})
}

// union renders an enum where any variant carries fields: a oneOf of one
// object per variant, each holding the discriminant with the variant's name
// as its value. It returns the definitions the variants take, in a dialect
// that names them.
func (b *Builder) union(s *Object, d *ir.Decl, name string) ([]def, error) {
	owner := emit.LastSegment(d.GetMeta().GetName())
	if b.Dialect.Unions == UnionsNone {
		return nil, emit.Unsupported(d.GetMeta().GetPosition(),
			"%s has variants carrying fields, and %s has no oneOf to write them with", owner, b.Dialect.Name)
	}
	disc := b.Discriminant
	if t, ok := b.Text(d.GetDirectives(), "discriminant"); ok {
		disc = t
	}
	closed := b.Closed || b.Tagged(d.GetDirectives(), "closed")

	var members []any
	var defs []def
	mapping := NewObject()
	taken := map[string]bool{name: true}
	for _, v := range d.GetEnumeration().GetVariants() {
		tag := v.GetMeta().GetName()
		m := NewObject()
		if b.Dialect.Unions == UnionsInline {
			m.Set("title", tag)
		}
		b.describe(m, v.GetMeta())
		if err := b.object(m, owner+"."+tag, disc, tag, v.GetFields(), closed); err != nil {
			return nil, err
		}
		if b.Dialect.Unions == UnionsInline {
			members = append(members, m)
			continue
		}

		vn := name + emit.Pascal(tag)
		if n, ok := b.Text(v.GetDirectives(), "name"); ok {
			vn = n
		}
		if err := b.free(v.GetMeta().GetPosition(), owner+"."+tag, vn); err != nil {
			return nil, err
		}
		if taken[vn] {
			return nil, emit.Unsupported(v.GetMeta().GetPosition(), "%s would define %s twice", owner, vn)
		}
		taken[vn] = true
		defs = append(defs, def{vn, m})
		members = append(members, NewObject().Set("$ref", b.Dialect.Ref+vn))
		mapping.Set(tag, b.Dialect.Ref+vn)
	}
	s.Set("oneOf", members)
	if b.Dialect.Unions == UnionsReferenced {
		s.Set("discriminator", NewObject().Set("propertyName", disc).Set("mapping", mapping))
	}
	return defs, nil
}

// property renders a field's schema and reports whether its key may be
// absent. A `T | null` field is required and may hold null; a `T?` one may
// be left out.
func (b *Builder) property(owner string, f *ir.Field) (*Object, bool, error) {
	r, err := b.Resolve(f.GetType())
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

	s, err := b.typ(r)
	if err != nil {
		return nil, false, err
	}
	if len(f.GetConstraints()) > 0 {
		b.constrain(s, b.kindOf(r), owner+"."+f.GetMeta().GetName(), f.GetConstraints())
	}
	if nullable {
		s = b.orNull(s)
	}

	out := NewObject()
	b.describe(out, f.GetMeta())
	for _, k := range s.keys {
		out.Set(k, s.vals[k])
	}
	return out, optional, nil
}

// typ is the schema for a reference. An optional value inside a collection
// is one that may be null, since JSON has no absent element.
func (b *Builder) typ(r *emit.Ref) (*Object, error) {
	switch r.Form {
	case emit.Prim:
		sc, ok := scalars[r.Name]
		if !ok {
			return nil, emit.Unsupported(r.Pos, "primitive %s has no %s type", r.Name, b.Dialect.Name)
		}
		s := NewObject().Set("type", sc.typ)
		switch {
		case sc.format != "":
			s.Set("format", sc.format)
		case b.Dialect.Formats && sc.width != "":
			s.Set("format", sc.width)
		}
		if sc.encoding != "" {
			if b.Dialect.ContentEncoding {
				s.Set("contentEncoding", sc.encoding)
			} else {
				s.Set("format", "byte")
			}
		}
		// An int32 format already says its bounds.
		if sc.bounded && (!b.Dialect.Formats || sc.width != "int32") {
			s.Set("minimum", sc.min).Set("maximum", sc.max)
		}
		if sc.unsigned {
			s.Set("minimum", int64(0))
		}
		return s, nil
	case emit.List, emit.Set:
		elem, err := b.typ(r.Elem)
		if err != nil {
			return nil, err
		}
		s := NewObject().Set("type", "array").Set("items", elem)
		if r.Form == emit.Set {
			s.Set("uniqueItems", true)
		}
		return s, nil
	case emit.Map:
		return b.dict(r)
	case emit.Option, emit.Nullable:
		inner, err := b.typ(r.Elem)
		if err != nil {
			return nil, err
		}
		return b.orNull(inner), nil
	}
	return NewObject().Set("$ref", b.Ref(r.Decl)), nil
}

// dict renders a map as an object whose values are the map's. A JSON
// object's keys are strings, so a key is anything that is a string in
// JSON, an integer, written as its digits, or a fieldless enum.
func (b *Builder) dict(r *emit.Ref) (*Object, error) {
	val, err := b.typ(r.Elem)
	if err != nil {
		return nil, err
	}
	s := NewObject().Set("type", "object")

	kx, err := b.Expand(r.Key)
	if err != nil {
		return nil, err
	}
	var keys *Object
	switch {
	case kx.Form == emit.Prim && scalars[kx.Name].typ == "string":
		if r.Key.Form == emit.Named {
			keys = NewObject().Set("$ref", b.Ref(r.Key.Decl))
		}
	case kx.Form == emit.Prim && scalars[kx.Name].typ == "integer":
		keys = NewObject().Set("pattern", `^-?(0|[1-9][0-9]*)$`)
	case kx.Form == emit.Named && kx.Decl.GetEnumeration() != nil && !emit.Fielded(kx.Decl.GetEnumeration()):
		keys = NewObject().Set("$ref", b.Ref(kx.Decl))
	default:
		return nil, emit.Unsupported(r.Pos, "a JSON object key is a string, an integer, or a fieldless enum, and this one is not")
	}
	if keys != nil && b.Dialect.PropertyNames {
		s.Set("propertyNames", keys)
	}
	s.Set("additionalProperties", val)
	return s, nil
}

// orNull is a schema that also accepts null.
func (b *Builder) orNull(s *Object) *Object {
	switch b.Dialect.Null {
	case NullNullable:
		return s.Set("nullable", true)
	case NullExtension:
		return s.Set("x-nullable", true)
	}
	if anyOf, ok := s.Get("anyOf"); ok && len(s.keys) == 1 {
		for _, m := range anyOf.([]any) {
			if t, _ := m.(*Object).Get("type"); t == "null" {
				return s
			}
		}
	}
	return NewObject().Set("anyOf", []any{s, NewObject().Set("type", "null")})
}

// Ref is the JSON pointer to a declaration's definition.
func (b *Builder) Ref(d *ir.Decl) string {
	return b.Dialect.Ref + b.DeclName(d, emit.Pascal)
}

// describe writes a node's documentation, and its deprecation, which a
// dialect without the keyword says in the description.
func (b *Builder) describe(s *Object, meta *ir.Meta) {
	lines := emit.Doc(meta)
	reason, deprecated := emit.Deprecated(meta)
	if deprecated && !b.Dialect.Deprecated {
		if len(lines) > 0 {
			lines = append(lines, "")
		}
		lines = append(lines, strings.TrimSpace("Deprecated. "+reason))
	}
	if len(lines) > 0 {
		s.Set("description", strings.Join(lines, "\n"))
	}
	if deprecated && b.Dialect.Deprecated {
		s.Set("deprecated", true)
	}
}

// Tagged reports whether a node carries an argument-less directive.
func (b *Builder) Tagged(all []*ir.Directive, name string) bool {
	for _, d := range plugin.Directives(b.Target, all) {
		if d.GetName() == name {
			return true
		}
	}
	return false
}

// BlockTagged reports whether the target block carries an argument-less
// directive.
func (b *Builder) BlockTagged(name string) bool {
	for _, block := range b.Model.GetTargets() {
		if block.GetMeta().GetName() == b.Target && b.Tagged(block.GetDirectives(), name) {
			return true
		}
	}
	return false
}

// Hoist moves every $ref with siblings into an allOf, for a dialect that
// ignores a $ref's siblings.
func Hoist(v any) {
	switch v := v.(type) {
	case *Object:
		for _, k := range v.keys {
			Hoist(v.vals[k])
		}
		ref, ok := v.Get("$ref")
		if !ok || len(v.keys) == 1 {
			return
		}
		v.del("$ref")
		v.Set("allOf", []any{NewObject().Set("$ref", ref)})
	case []any:
		for _, e := range v {
			Hoist(e)
		}
	}
}

func asWritten(name string) string { return name }
