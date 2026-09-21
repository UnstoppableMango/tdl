package salesforce

import (
	"encoding/xml"
	"fmt"
	"path"
	"strings"

	"github.com/unstoppablemango/tdl/backend/internal/emit"
	"github.com/unstoppablemango/tdl/ir"
	"github.com/unstoppablemango/tdl/plugin"
)

// apexTypes maps a TDL primitive to its Apex type. Apex has no UUID or
// duration, so each is the string it would be in JSON.
var apexTypes = map[string]string{
	"string":   "String",
	"int":      "Long",
	"bool":     "Boolean",
	"bytes":    "Blob",
	"decimal":  "Decimal",
	"uuid":     "String",
	"instant":  "Datetime",
	"date":     "Date",
	"duration": "String",
}

// reserved is the words Apex refuses as an identifier, in lower case since
// Apex reads identifiers without case.
var reserved = map[string]bool{}

// standard is the types an org already declares that a model is likely to
// name too. An Apex class named like one shadows it in every class that
// could have used it.
var standard = map[string]bool{}

func init() {
	for _, w := range strings.Fields(`
		abstract activate and any array as asc autonomous begin bigdecimal blob
		boolean break bulk by byte case cast catch char class collect commit
		const continue currency date datetime decimal default delete desc do
		double else end enum exception exit export extends false final finally
		float for from global goto group having hint if implements import in
		inner insert instanceof int integer interface into join like limit list
		long loop map merge new not null nulls number object of on or outer
		override package parallel pragma private protected public retrieve
		return returning rollback savepoint search select set short sort stat
		static string super switch synchronized system testmethod then this
		throw time transaction trigger true try type undelete update upsert
		using virtual webservice when where while`) {
		reserved[w] = true
	}
	for _, w := range strings.Fields(`
		Account Asset Campaign Case Contact Contract Database Event Id JSON
		Lead Limits Math Opportunity Order Pricebook2 Product2 Quote Schema
		Task Test User`) {
		standard[strings.ToLower(w)] = true
	}
}

// ident says why n cannot be an Apex identifier, if it cannot.
func ident(pos *ir.Position, what, n string) error {
	if err := checkName(pos, what, n); err != nil {
		return err
	}
	if reserved[strings.ToLower(n)] {
		return emit.Unsupported(pos, "%s would be named %s in Apex, which is a reserved word", what, n)
	}
	return nil
}

// className is the Apex name of a value, a mixin, or an enum.
func (g *generator) className(d *ir.Decl) string {
	return g.prefix + g.DeclName(d, emit.Pascal)
}

// declare checks and claims the Apex name a declaration takes.
func (g *generator) declare(d *ir.Decl) (string, error) {
	pos := d.GetMeta().GetPosition()
	name := d.GetMeta().GetName()
	n := g.className(d)
	if err := ident(pos, name, n); err != nil {
		return "", err
	}
	if standard[strings.ToLower(n)] {
		return "", emit.Unsupported(pos,
			"%s would be named %s in Apex, which shadows the standard type; a name directive or a prefix renames it", name, n)
	}
	return n, claim(g.classes, pos, name, n)
}

// class renders a value or a mixin as an Apex class with a public member
// per field. Members keep their TDL names, so JSON.serialize writes what
// the other wire backends describe.
func (g *generator) class(d *ir.Decl) ([]*plugin.File, error) {
	n, err := g.declare(d)
	if err != nil {
		return nil, err
	}
	var b strings.Builder
	comment(&b, "", d.GetMeta())
	fmt.Fprintf(&b, "public class %s {\n", n)
	if err := g.members(&b, "    ", d.GetMeta().GetName(), d.Fields()); err != nil {
		return nil, err
	}
	b.WriteString("}\n")
	return g.apex(n, b.String()), nil
}

// enum renders a fieldless enum. Values keep their TDL names, which are
// also the picklist values an object field holding the enum accepts.
func (g *generator) enum(d *ir.Decl) ([]*plugin.File, error) {
	n, err := g.declare(d)
	if err != nil {
		return nil, err
	}
	var b strings.Builder
	comment(&b, "", d.GetMeta())
	fmt.Fprintf(&b, "public enum %s {\n", n)
	if err := g.variants(&b, "    ", d); err != nil {
		return nil, err
	}
	b.WriteString("}\n")
	return g.apex(n, b.String()), nil
}

// sum renders an enum whose variants carry fields. Apex has no union, so it
// is a class holding a `kind` naming the variant and a member per variant
// with fields, each an inner class; the member for the kind not chosen is
// null. That is a protobuf oneof's shape.
func (g *generator) sum(d *ir.Decl) ([]*plugin.File, error) {
	n, err := g.declare(d)
	if err != nil {
		return nil, err
	}
	name := d.GetMeta().GetName()

	var b strings.Builder
	comment(&b, "", d.GetMeta())
	fmt.Fprintf(&b, "public class %s {\n", n)
	b.WriteString("    public enum Kind {\n")
	if err := g.variants(&b, "        ", d); err != nil {
		return nil, err
	}
	b.WriteString("    }\n\n    public Kind kind;\n")

	taken := map[string]string{"kind": "the discriminant", strings.ToLower(n): name}
	var inner strings.Builder
	for _, v := range d.GetEnumeration().GetVariants() {
		if len(v.GetFields()) == 0 {
			continue
		}
		pos := v.GetMeta().GetPosition()
		vn := v.GetMeta().GetName()
		cls := emit.Pascal(vn)
		if s, ok := g.Text(v.GetDirectives(), "name"); ok {
			cls = s
		}
		member := emit.Camel(cls)
		if err := ident(pos, name+"."+vn, cls); err != nil {
			return nil, err
		}
		// A member may share its type's name, as `Card card` does, so one
		// check covers both: Kind, the class itself, and each variant.
		if other, ok := taken[strings.ToLower(cls)]; ok {
			return nil, emit.Unsupported(pos, "%s.%s would be named %s in Apex, which %s already is", name, vn, cls, other)
		}
		taken[strings.ToLower(cls)] = vn

		comment(&b, "    ", v.GetMeta())
		fmt.Fprintf(&b, "    public %s %s;\n", cls, member)
		fmt.Fprintf(&inner, "\n    public class %s {\n", cls)
		if err := g.members(&inner, "        ", name+"."+vn, v.GetFields()); err != nil {
			return nil, err
		}
		inner.WriteString("    }\n")
	}
	b.WriteString(inner.String())
	b.WriteString("}\n")
	return g.apex(n, b.String()), nil
}

// variants writes an enum's values, one per line.
func (g *generator) variants(b *strings.Builder, indent string, d *ir.Decl) error {
	seen := map[string]bool{}
	vs := d.GetEnumeration().GetVariants()
	for i, v := range vs {
		vn := v.GetMeta().GetName()
		if err := ident(v.GetMeta().GetPosition(), d.GetMeta().GetName()+"."+vn, vn); err != nil {
			return err
		}
		if seen[strings.ToLower(vn)] {
			return emit.Unsupported(v.GetMeta().GetPosition(), "%s has two values named %s in Apex", d.GetMeta().GetName(), vn)
		}
		seen[strings.ToLower(vn)] = true
		comment(b, indent, v.GetMeta())
		sep := ","
		if i == len(vs)-1 {
			sep = ""
		}
		fmt.Fprintf(b, "%s%s%s\n", indent, vn, sep)
	}
	return nil
}

// members writes a public member per field.
func (g *generator) members(b *strings.Builder, indent, owner string, fields []*ir.Field) error {
	seen := map[string]bool{}
	for _, f := range fields {
		pos := f.GetMeta().GetPosition()
		what := owner + "." + f.GetMeta().GetName()
		n := g.FieldName(f, asWritten)
		if err := ident(pos, what, n); err != nil {
			return err
		}
		if seen[strings.ToLower(n)] {
			return emit.Unsupported(pos, "%s has two members named %s in Apex, which reads names without case", owner, n)
		}
		seen[strings.ToLower(n)] = true

		ref, err := g.Resolve(f.GetType())
		if err != nil {
			return err
		}
		typ, err := g.apexType(ref)
		if err != nil {
			return err
		}
		comment(b, indent, f.GetMeta())
		fmt.Fprintf(b, "%spublic %s %s;\n", indent, typ, n)
	}
	return nil
}

// apexType is the Apex type for a reference. Every Apex variable may be
// null, so an optional type is the type it wraps.
func (g *generator) apexType(r *emit.Ref) (string, error) {
	r, err := g.Expand(r)
	if err != nil {
		return "", err
	}
	switch r.Form {
	case emit.Prim:
		typ, ok := apexTypes[r.Name]
		if !ok {
			return "", emit.Unsupported(r.Pos, "primitive %s has no Apex type", r.Name)
		}
		return typ, nil
	case emit.Option, emit.Nullable:
		return g.apexType(r.Elem)
	case emit.List, emit.Set:
		elem, err := g.apexType(r.Elem)
		if err != nil {
			return "", err
		}
		if r.Form == emit.Set {
			return "Set<" + elem + ">", nil
		}
		return "List<" + elem + ">", nil
	case emit.Map:
		key, err := g.apexType(r.Key)
		if err != nil {
			return "", err
		}
		val, err := g.apexType(r.Elem)
		if err != nil {
			return "", err
		}
		return "Map<" + key + ", " + val + ">", nil
	}
	if r.Decl.GetStructure().GetKind() == ir.StructKind_STRUCT_KIND_ENTITY {
		return g.objectName(r.Decl), nil
	}
	return g.className(r.Decl), nil
}

// apex is a class's source and the metadata file every class carries.
func (g *generator) apex(n, src string) []*plugin.File {
	meta := &struct {
		XMLName    xml.Name `xml:"ApexClass"`
		NS         string   `xml:"xmlns,attr"`
		APIVersion string   `xml:"apiVersion"`
		Status     string   `xml:"status"`
	}{NS: metadataNS, APIVersion: g.apiVersion, Status: "Active"}

	p := path.Join("classes", n+".cls")
	return []*plugin.File{
		{Path: p, Content: []byte("// Code generated by tdl. DO NOT EDIT.\n\n" + src)},
		xmlFile(p+"-meta.xml", meta),
	}
}

func asWritten(name string) string { return name }

// comment writes a node's documentation, and its deprecation, as ApexDoc.
// An @Deprecated annotation is only legal in a managed package, so the
// deprecation is a tag in the comment instead.
func comment(b *strings.Builder, indent string, meta *ir.Meta) {
	lines := emit.Doc(meta)
	if reason, ok := emit.Deprecated(meta); ok {
		if len(lines) > 0 {
			lines = append(lines, "")
		}
		lines = append(lines, strings.TrimSpace("@deprecated "+reason))
	}
	if len(lines) == 0 {
		return
	}
	fmt.Fprintf(b, "%s/**\n", indent)
	for _, line := range lines {
		if line == "" {
			fmt.Fprintf(b, "%s *\n", indent)
			continue
		}
		fmt.Fprintf(b, "%s * %s\n", indent, strings.ReplaceAll(line, "*/", "* /"))
	}
	fmt.Fprintf(b, "%s */\n", indent)
}
