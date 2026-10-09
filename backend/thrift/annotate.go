package thrift

import (
	"cmp"
	"fmt"
	"slices"
	"strings"

	"github.com/unstoppablemango/tdl/ast"
	"github.com/unstoppablemango/tdl/backend/internal/emit"
	"github.com/unstoppablemango/tdl/backend/internal/reverse"
	"github.com/unstoppablemango/tdl/ir"
)

// carrier is the namespace language, and name, that holds the file's
// annotations when the model has no package to write as `namespace *`.
const carrier = "tdl"

// kv is one tdl.* annotation.
type kv struct{ key, value string }

// annotations is a node's parenthesized annotations: its deprecation, then
// the tdl.* ones when roundtrip is on, or "".
func (g *generator) annotations(meta *ir.Meta, kvs ...kv) string {
	var parts []string
	if reason, ok := emit.Deprecated(meta); ok {
		parts = append(parts, "deprecated = "+quote(reason))
	}
	if g.Roundtrip {
		for _, a := range kvs {
			parts = append(parts, a.key+" = "+literal(a.value))
		}
	}
	if len(parts) == 0 {
		return ""
	}
	return " (" + strings.Join(parts, ", ") + ")"
}

// literal writes an annotation value as a Thrift string literal that
// thriftgo reads back exactly. thriftgo unescapes only `\"`, keeps `\\`
// as two characters, and ends a literal at a `"` a backslash does not
// escape, so a backslash before a quote or at the end is written %5C, and
// a percent sign %25. Newlines stay raw.
func literal(s string) string {
	var b strings.Builder
	b.WriteByte('"')
	for i := 0; i < len(s); i++ {
		switch c := s[i]; c {
		case '%':
			b.WriteString("%25")
		case '"':
			b.WriteString(`\"`)
		case '\\':
			j := i
			for j < len(s) && s[j] == '\\' {
				j++
			}
			if j == len(s) || s[j] == '"' {
				b.WriteString(strings.Repeat("%5C", j-i))
			} else {
				b.WriteString(s[i:j])
			}
			i = j - 1
		default:
			b.WriteByte(c)
		}
	}
	b.WriteByte('"')
	return b.String()
}

var unliteral = strings.NewReplacer("%25", "%", "%5C", `\`)

// nameKVs warns when a declaration's Thrift name does not read back as its
// own, and carries it.
func (g *generator) nameKVs(d *ir.Decl, written string) []kv {
	name := d.GetMeta().GetName()
	if written != name {
		g.Lose(emit.LossName, d.GetMeta().GetPosition(), "%s is written %s, which reads back as its name", name, written)
		return []kv{{"tdl.name", name}}
	}
	g.CheckRenamed(d.GetDirectives(), emit.Pascal(written) != written, "%s", name)
	return nil
}

// structKVs is a struct's annotations: its name, its kind, since every
// struct reads back as a value, and its conformances.
func (g *generator) structKVs(d *ir.Decl, names []string) []kv {
	if len(names) == 0 {
		return nil
	}
	kvs := g.nameKVs(d, names[0])
	name := d.GetMeta().GetName()
	switch d.GetStructure().GetKind() {
	case ir.StructKind_STRUCT_KIND_ENTITY:
		g.Lose(emit.LossStructKind, d.GetMeta().GetPosition(), "%s is an entity, and reads back as a value", name)
		kvs = append(kvs, kv{"tdl.kind", "entity"})
	case ir.StructKind_STRUCT_KIND_MIXIN:
		g.Lose(emit.LossStructKind, d.GetMeta().GetPosition(), "%s is a mixin, and reads back as a value", name)
		kvs = append(kvs, kv{"tdl.kind", "mixin"})
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
			g.Lose(emit.LossClass, c.GetPosition(), "%s conforms to %s, which Thrift does not carry", name, n)
			lost = true
		}
	}
	if !lost {
		return nil
	}
	return []kv{{"tdl.conforms", g.Conforms(name)}}
}

// docKVs carries a node's doc comment and deprecation reason when the
// comment Generate writes does not read back as them.
func (g *generator) docKVs(meta *ir.Meta) []kv {
	var b strings.Builder
	comment(&b, "", meta)
	_, deprecated := emit.Deprecated(meta)
	h := reverse.Head("", docLines(strings.TrimSuffix(b.String(), "\n")), ast.Position{}, deprecated)
	reason, _ := emit.Deprecated(meta)
	if slices.Equal(h.Doc, meta.GetDoc()) && (h.Dep == nil || h.Dep.Reason == reason) {
		return nil
	}
	g.Lose(emit.LossDoc, meta.GetPosition(), "%s's doc comment does not read back from a Thrift comment", meta.GetName())
	kvs := []kv{{"tdl.doc", strings.Join(meta.GetDoc(), "\n")}}
	if deprecated {
		kvs = append(kvs, kv{"tdl.reason", reason})
	}
	return kvs
}

// fieldKVs warns about and carries what a field loses: how it is written,
// what Thrift has no form for, and the include that copied it.
func (g *generator) fieldKVs(owner string, f *ir.Field, ref *emit.Ref, written, include string) []kv {
	name := f.GetMeta().GetName()
	pos := f.GetMeta().GetPosition()
	if written != name {
		g.Lose(emit.LossName, pos, "%s.%s is written %s, which reads back as its name", owner, name, written)
	} else {
		g.CheckRenamed(f.GetDirectives(), false, "%s.%s", owner, name)
	}
	if f.GetOwned() {
		g.Lose(emit.LossOwned, pos, "%s.%s is owned, which Thrift does not carry", owner, name)
	}
	if f.GetDefaultValue() != nil {
		g.Lose(emit.LossDefault, pos, "%s.%s has a default, which is not written", owner, name)
	}
	g.typeLosses(f.GetType(), pos)

	var kvs []kv
	if src := g.FieldSource(f, false); src != written+": "+g.spell(ref) {
		kvs = append(kvs, kv{"tdl.source", src})
	}
	if include != "" {
		kvs = append(kvs, kv{"tdl.include", include})
	}
	return append(kvs, g.docKVs(f.GetMeta())...)
}

// lossyPrimitives is the primitives whose Thrift type reads back as
// another.
var lossyPrimitives = map[string]bool{
	"int": true, "uint32": true, "float32": true,
	"decimal": true, "uuid": true, "instant": true, "date": true, "duration": true,
}

// typeLosses warns about what a type loses in Thrift: a primitive's kind,
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
		case name == "List" && wrote != ir.SyntacticForm_SYNTACTIC_FORM_BRACKETS,
			name == "Set" && wrote != ir.SyntacticForm_SYNTACTIC_FORM_BRACES,
			name == "Map" && wrote != ir.SyntacticForm_SYNTACTIC_FORM_ARROW:
			g.Lose(emit.LossCollection, pos, "%s reads back written as sugar", name)
		case lossyPrimitives[name]:
			g.Lose(emit.LossPrimitive, pos, "%s is Thrift %s, which reads back as %s", name, scalars[name], primitiveOf(scalars[name]))
		}
	case decl.GetEnumeration() != nil && name == "Nullable":
		g.Lose(emit.LossOptional, pos, "a nullable value is an optional field in Thrift, and reads back as T?")
	case decl.GetEnumeration() != nil && name == "Option" && wrote != ir.SyntacticForm_SYNTACTIC_FORM_QUESTION:
		g.Lose(emit.LossOptional, pos, "Option reads back written as T?")
	}
	for _, arg := range t.GetArgs() {
		g.typeLosses(arg, at)
	}
}

// primitiveOf is the TDL primitive a Thrift base type reads back as.
func primitiveOf(base string) string {
	switch base {
	case "i64":
		return "int64"
	case "i32":
		return "int32"
	case "double":
		return "float64"
	case "binary":
		return "bytes"
	}
	return base
}

// spell is the TDL a field of type r reads back as, without annotations.
func (g *generator) spell(r *emit.Ref) string {
	switch r.Form {
	case emit.Option, emit.Nullable:
		return g.spell(r.Elem) + "?"
	case emit.List:
		return "[" + g.spell(r.Elem) + "]"
	case emit.Set:
		return "{" + g.spell(r.Elem) + "}"
	case emit.Map:
		return "{" + g.spell(r.Key) + " -> " + g.spell(r.Elem) + "}"
	case emit.Prim:
		return primitiveOf(scalars[r.Name])
	}
	return r.Decl.GetMeta().GetName()
}

// newtypeSource is a newtype as TDL, from its name on, without its doc
// comment or deprecation: `Email: string`.
func (g *generator) newtypeSource(d *ir.Decl) string {
	for _, decl := range g.Unlowered().Decls {
		n, ok := decl.(*ast.NewtypeDecl)
		if !ok || n.N != d.GetMeta().GetName() {
			continue
		}
		bare := *n
		bare.Doc, bare.DocP, bare.Dep = nil, nil, nil
		return strings.TrimPrefix(emit.PrintItem(&bare), "type ")
	}
	return ""
}

// fileAnnotation is the namespace's annotations: the TDL package when the
// namespace renames it, the package's doc comment, the imports, and every
// top-level item no Thrift definition declares, by its index in the file.
func (g *generator) fileAnnotation(emitted map[string]bool) string {
	if !g.Roundtrip {
		return ""
	}
	var kvs []kv
	if pkg := g.Model.GetPackage(); pkg != g.ns {
		kvs = append(kvs, kv{"tdl.package", pkg})
	}
	if doc := g.Model.GetDoc(); len(doc) > 0 {
		kvs = append(kvs, kv{"tdl.doc", strings.Join(doc, "\n")})
	}
	for _, imp := range g.Imports() {
		kvs = append(kvs, kv{"tdl.import", imp})
	}
	for _, it := range g.Carried(emitted) {
		kvs = append(kvs, kv{"tdl.item", fmt.Sprintf("%d %s", it.At, it.Source)})
	}
	if len(kvs) == 0 {
		return ""
	}
	parts := make([]string, len(kvs))
	for i, a := range kvs {
		parts[i] = "  " + a.key + " = " + literal(a.value)
	}
	return " (\n" + strings.Join(parts, "\n") + "\n)"
}

// docLines is the doc comment in the comments thriftgo keeps before a
// node: the last of them when it is a /** */ block, each line without its
// leading `*`.
func docLines(comments string) []string {
	i := strings.LastIndex(comments, "/**")
	if i < 0 || !strings.HasSuffix(comments, "*/") || len(comments)-i < len("/***/") {
		return nil
	}
	lines := strings.Split(comments[i+len("/**"):len(comments)-len("*/")], "\n")
	if len(lines) == 1 {
		if line := strings.TrimSpace(lines[0]); line != "" {
			return []string{line}
		}
		return nil
	}
	if strings.TrimSpace(lines[0]) == "" {
		lines = lines[1:]
	}
	if n := len(lines); n > 0 && strings.TrimSpace(lines[n-1]) == "" {
		lines = lines[:n-1]
	}
	out := make([]string, len(lines))
	for i, line := range lines {
		line = strings.TrimLeft(line, " \t")
		if rest, ok := strings.CutPrefix(line, "*"); ok {
			line = strings.TrimPrefix(rest, " ")
		}
		out[i] = strings.TrimRight(line, " \t\r")
	}
	return out
}
