package graphql

import (
	"cmp"
	"fmt"
	"slices"
	"strings"

	gqlast "github.com/vektah/gqlparser/v2/ast"
	gqlparser "github.com/vektah/gqlparser/v2/parser"

	"github.com/unstoppablemango/tdl/ast"
	"github.com/unstoppablemango/tdl/backend/internal/emit"
	"github.com/unstoppablemango/tdl/ir"
)

// directive is the @tdl directive's name, and definition its definition,
// written once in a schema that uses it.
const (
	directive  = "tdl"
	definition = `"""
What TDL needs to rebuild the model this schema was generated from.
"""
directive @tdl(name: String, kind: String, conforms: String, source: String, include: String, doc: String, reason: String, package: String, import: String, item: String) repeatable on SCHEMA | OBJECT | FIELD_DEFINITION | ENUM | ENUM_VALUE | UNION
`
)

// kv is one @tdl argument.
type kv struct{ key, value string }

// annotations is a node's @tdl directives when roundtrip is on, one per
// argument, or "".
func (g *generator) annotations(kvs []kv) string {
	if !g.Roundtrip || len(kvs) == 0 {
		return ""
	}
	g.annotated = true
	var b strings.Builder
	for _, a := range kvs {
		fmt.Fprintf(&b, " @%s(%s: %s)", directive, a.key, literal(a.value))
	}
	return b.String()
}

// literal writes s as a GraphQL string literal.
func literal(s string) string {
	var b strings.Builder
	b.WriteByte('"')
	for _, r := range s {
		switch r {
		case '"', '\\':
			b.WriteByte('\\')
			b.WriteRune(r)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '\t':
			b.WriteString(`\t`)
		default:
			if r < 0x20 || r == 0x7f {
				fmt.Fprintf(&b, `\u%04x`, r)
				continue
			}
			b.WriteRune(r)
		}
	}
	b.WriteByte('"')
	return b.String()
}

// nameKVs warns when a type's GraphQL name does not read back as its own,
// and carries it.
func (g *generator) nameKVs(d *ir.Decl, written string) []kv {
	name := d.GetMeta().GetName()
	if written != name {
		g.Lose(emit.LossName, d.GetMeta().GetPosition(), "%s is written %s, which reads back as its name", name, written)
		return []kv{{"name", name}}
	}
	g.CheckRenamed(d.GetDirectives(), emit.Pascal(written) != written, "%s", name)
	return nil
}

// structKVs is an object's annotations: its name, its kind, since every
// object reads back as a value, and its conformances.
func (g *generator) structKVs(d *ir.Decl, written string) []kv {
	kvs := g.nameKVs(d, written)
	name := d.GetMeta().GetName()
	switch d.GetStructure().GetKind() {
	case ir.StructKind_STRUCT_KIND_ENTITY:
		g.Lose(emit.LossStructKind, d.GetMeta().GetPosition(), "%s is an entity, and reads back as a value", name)
		kvs = append(kvs, kv{"kind", "entity"})
	case ir.StructKind_STRUCT_KIND_MIXIN:
		g.Lose(emit.LossStructKind, d.GetMeta().GetPosition(), "%s is a mixin, and reads back as a value", name)
		kvs = append(kvs, kv{"kind", "mixin"})
	}
	return append(kvs, g.conformsKV(d)...)
}

// conformsKV carries a declaration's conformances, other than a struct's
// to Entity, which its kind says.
func (g *generator) conformsKV(d *ir.Decl) []kv {
	conforms := d.GetStructure().GetConforms()
	if e := d.GetEnumeration(); e != nil {
		conforms = e.GetConforms()
	}
	name := d.GetMeta().GetName()
	lost := false
	for _, c := range conforms {
		if n := c.GetClass().GetName(); d.GetStructure() == nil || n != "Entity" || c.GetExtern() != nil || len(c.GetArgs()) > 0 {
			g.Lose(emit.LossClass, c.GetPosition(), "%s conforms to %s, which GraphQL does not carry", name, n)
			lost = true
		}
	}
	if !lost {
		return nil
	}
	return []kv{{"conforms", g.Conforms(name)}}
}

// node is what a description belongs to, which decides how import reads
// it.
type node int

const (
	// member is a field or an enum value, deprecated by @deprecated.
	member node = iota
	// object is a type, whose description ends in its deprecation.
	object
	// empty is a fieldless variant's object, whose description ends in
	// the placeholder's note.
	empty
)

// docKVs carries a node's doc comment, and a type's deprecation reason,
// when the description written as lines does not read back as them.
func (g *generator) docKVs(meta *ir.Meta, kind node, indent string, lines []string) []kv {
	doc, dep := readDescription(blockValue(indent, lines), kind)
	reason, deprecated := emit.Deprecated(meta)
	same := slices.Equal(doc, meta.GetDoc())
	if kind != member {
		same = same && (dep != nil) == deprecated && (dep == nil || dep.Reason == reason)
	}
	if same {
		return nil
	}
	g.Lose(emit.LossDoc, meta.GetPosition(), "%s's doc comment does not read back from a GraphQL description", meta.GetName())
	kvs := []kv{{"doc", strings.Join(meta.GetDoc(), "\n")}}
	if deprecated {
		kvs = append(kvs, kv{"reason", reason})
	}
	return kvs
}

// blockValue is the string a description written as lines holds, as
// gqlparser reads it.
func blockValue(indent string, lines []string) string {
	if len(lines) == 0 {
		return ""
	}
	var b strings.Builder
	description(&b, indent, lines)
	b.WriteString("scalar X\n")
	doc, err := gqlparser.ParseSchema(&gqlast.Source{Input: b.String()})
	if err != nil || len(doc.Definitions) == 0 {
		return ""
	}
	return doc.Definitions[0].Description
}

// placeholderNote ends the description of a fieldless variant's object.
func placeholderNote(variant string) string {
	return fmt.Sprintf("The %s variant, which carries nothing. %s", variant, placeholderTail)
}

const placeholderTail = "`" + placeholder + "` is always null: a GraphQL object needs at least one field."

// readDescription is the doc comment and deprecation import reads from a
// description: a type's last paragraph is its deprecation, after a
// fieldless variant's note is set aside.
func readDescription(desc string, kind node) ([]string, *ast.Deprecation) {
	var lines []string
	if desc != "" {
		lines = strings.Split(desc, "\n")
	}
	last := func() string { return lines[len(lines)-1] }
	drop := func() {
		lines = lines[:len(lines)-1]
		if n := len(lines); n > 0 && lines[n-1] == "" {
			lines = lines[:n-1]
		}
	}
	if kind == empty && len(lines) > 0 && strings.HasPrefix(last(), "The ") && strings.HasSuffix(last(), placeholderTail) {
		drop()
	}
	if kind == member || len(lines) == 0 {
		return lines, nil
	}
	if last() == "Deprecated." {
		drop()
		return lines, &ast.Deprecation{}
	}
	if reason, ok := strings.CutPrefix(last(), "Deprecated: "); ok {
		drop()
		return lines, &ast.Deprecation{Reason: reason}
	}
	return lines, nil
}

// fieldKVs warns about and carries what a field loses: how it is written,
// what GraphQL has no form for, and the include that copied it.
func (g *generator) fieldKVs(owner string, f *ir.Field, ref *emit.Ref, written, include string) []kv {
	name := f.GetMeta().GetName()
	pos := f.GetMeta().GetPosition()
	if written != name {
		g.Lose(emit.LossName, pos, "%s.%s is written %s, which reads back as its name", owner, name, written)
	} else {
		g.CheckRenamed(f.GetDirectives(), false, "%s.%s", owner, name)
	}
	if f.GetOwned() {
		g.Lose(emit.LossOwned, pos, "%s.%s is owned, which GraphQL does not carry", owner, name)
	}
	if f.GetDefaultValue() != nil {
		g.Lose(emit.LossDefault, pos, "%s.%s has a default, which is not written", owner, name)
	}
	g.typeLosses(f.GetType(), pos)

	var kvs []kv
	if src := g.FieldSource(f, false); src != written+": "+g.spell(ref) {
		kvs = append(kvs, kv{"source", src})
	}
	if include != "" {
		kvs = append(kvs, kv{"include", include})
	}
	return kvs
}

// lossyPrimitives is the primitives whose GraphQL scalar reads back as
// another.
var lossyPrimitives = map[string]bool{"int": true, "uint32": true, "float32": true}

// typeLosses warns about what a type loses in GraphQL: a primitive's kind,
// and sugar import writes another way. A newtype or alias it names warns
// where it is declared.
func (g *generator) typeLosses(id *ir.ID, at *ir.Position) {
	t := g.Model.Type(id)
	if t.GetUnit() != nil {
		g.Lose(emit.LossUnit, cmp.Or(at, t.GetPosition()), "the unit %s is not carried", t.GetUnit().GetName())
		return
	}
	decl := g.Model.Decl(t.GetCtor())
	if decl == nil || decl.GetAlias() != nil || decl.GetNewtype() != nil {
		return
	}
	// A type is interned at its first use, so a field's own position says
	// which use is meant.
	pos := cmp.Or(at, t.GetPosition())
	name := decl.GetMeta().GetName()
	wrote := t.GetWrote()
	switch {
	case decl.GetPrimitive() != nil:
		switch {
		case name == "Set":
			g.Lose(emit.LossCollection, pos, "a set is a GraphQL list, and reads back as one")
		case name == "List" && wrote != ir.SyntacticForm_SYNTACTIC_FORM_BRACKETS:
			g.Lose(emit.LossCollection, pos, "List reads back written as sugar")
		case lossyPrimitives[name]:
			g.Lose(emit.LossPrimitive, pos, "%s is GraphQL %s, which reads back as %s", name, scalars[name], primitives[scalars[name]])
		}
	case decl.GetEnumeration() != nil && name == "Nullable":
		g.Lose(emit.LossOptional, pos, "a nullable value is a nullable GraphQL type, and reads back as T?")
	case decl.GetEnumeration() != nil && name == "Option" && wrote != ir.SyntacticForm_SYNTACTIC_FORM_QUESTION:
		g.Lose(emit.LossOptional, pos, "Option reads back written as T?")
	}
	for _, arg := range t.GetArgs() {
		g.typeLosses(arg, at)
	}
}

// spell is the TDL a field of type r reads back as, without annotations.
func (g *generator) spell(r *emit.Ref) string {
	if expanded, err := g.Expand(r); err == nil {
		r = expanded
	}
	switch r.Form {
	case emit.Option, emit.Nullable:
		return g.spell(r.Elem) + "?"
	case emit.List, emit.Set:
		return "[" + g.spell(r.Elem) + "]"
	case emit.Prim:
		return primitives[scalars[r.Name]]
	}
	return r.Decl.GetMeta().GetName()
}

// packageBack is the package a model's file reads back as: the name of the
// file, unless it is the one a model without a package is written to.
func packageBack(pkg string) string {
	if pkg == "" {
		return ""
	}
	if last := emit.LastSegment(pkg); last != defaultFile {
		return last
	}
	return ""
}

// schemaExtension is `extend schema` carrying the TDL package when the file
// name does not, the package's doc comment, the imports, and every
// top-level item no GraphQL type declares, by its index in the file; or "".
func (g *generator) schemaExtension(emitted map[string]bool) string {
	if !g.Roundtrip {
		return ""
	}
	var kvs []kv
	if pkg := g.Model.GetPackage(); packageBack(pkg) != pkg {
		kvs = append(kvs, kv{"package", pkg})
	}
	if doc := g.Model.GetDoc(); len(doc) > 0 {
		kvs = append(kvs, kv{"doc", strings.Join(doc, "\n")})
	}
	for _, imp := range g.Imports() {
		kvs = append(kvs, kv{"import", imp})
	}
	for _, it := range g.Carried(emitted) {
		kvs = append(kvs, kv{"item", fmt.Sprintf("%d %s", it.At, it.Source)})
	}
	if len(kvs) == 0 {
		return ""
	}
	g.annotated = true
	var b strings.Builder
	b.WriteString("extend schema")
	for _, a := range kvs {
		fmt.Fprintf(&b, "\n  @%s(%s: %s)", directive, a.key, literal(a.value))
	}
	b.WriteString("\n")
	return b.String()
}
