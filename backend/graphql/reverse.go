package graphql

import (
	"bytes"
	"cmp"
	"context"
	"fmt"
	"path"
	"slices"
	"strings"

	"github.com/vektah/gqlparser/v2"
	gqlast "github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/formatter"
	gqlparse "github.com/vektah/gqlparser/v2/parser"

	"github.com/unstoppablemango/tdl/ast"
	"github.com/unstoppablemango/tdl/backend/internal/emit"
	"github.com/unstoppablemango/tdl/backend/internal/reverse"
	"github.com/unstoppablemango/tdl/ir"
	"github.com/unstoppablemango/tdl/plugin"
)

// primitives maps a scalar to the TDL primitive it reads as: the inverse of
// scalars, with ID read as a string.
var primitives = map[string]string{
	"String":   "string",
	"Int":      "int32",
	"Long":     "int64",
	"UInt64":   "uint64",
	"Float":    "float64",
	"Boolean":  "bool",
	"Bytes":    "bytes",
	"Decimal":  "decimal",
	"UUID":     "uuid",
	"DateTime": "instant",
	"Date":     "date",
	"Duration": "duration",
	"ID":       "string",
}

// Import reads one .graphql schema into a model: each object type a
// struct, each enum an enum, and a union of object types nothing else
// names, as Generate writes a fielded enum, an enum with fields. The file's
// name is the package. What regenerating needs goes in a graphql target
// block. Under the @tdl directives a `roundtrip` directive writes, it
// rebuilds the model the schema was generated from.
// docs/design/schema-backends.md has the mapping.
func (Backend) Import(_ context.Context, req *plugin.ImportRequest) (*plugin.ImportResponse, error) {
	r := &reader{names: map[string]string{}, sums: map[string]bool{}, variants: map[string]bool{}}
	return r.Response(r.read(req.GetFiles()))
}

type reader struct {
	reverse.Reader
	path string
	doc  *gqlast.SchemaDocument

	// names maps each GraphQL type import reads to its TDL name.
	names map[string]string
	// sums is the unions read as enums, and variants the object types they
	// absorb.
	sums     map[string]bool
	variants map[string]bool
}

func (r *reader) read(files []*plugin.File) (*ir.Model, error) {
	if len(files) != 1 {
		return nil, reverse.Failf(nil, "import reads one .graphql file, and was given %d", len(files))
	}
	r.path = files[0].GetPath()
	src := &gqlast.Source{Name: r.path, Input: string(files[0].GetContent())}
	doc, err := gqlparse.ParseSchema(src)
	if err != nil {
		return nil, reverse.Failf(r.pos(nil), "%s does not parse: %v", r.path, err)
	}
	if _, err := gqlparser.LoadSchema(src); err != nil {
		return nil, reverse.Failf(r.pos(nil), "%s is not a valid schema: %v", r.path, err)
	}
	r.doc = doc
	r.Roundtrip = slices.ContainsFunc(doc.Directives, func(d *gqlast.DirectiveDefinition) bool { return d.Name == directive })

	file := &ast.File{Filename: r.path}
	pkg, items, pdoc, err := r.header(file)
	if err != nil {
		return nil, err
	}
	if pkg != "" || len(pdoc) > 0 {
		file.Package = &ast.PackageDecl{Path: pkg, Doc: pdoc, DocP: make([]ast.Position, len(pdoc))}
	}

	r.nameAll()
	var emitted []reverse.Item
	for _, def := range doc.Definitions {
		decl, err := r.definition(def)
		if err != nil {
			return nil, err
		}
		if decl != nil {
			emitted = append(emitted, reverse.Item{Decl: decl, At: -1})
		}
	}
	file.Decls = reverse.Arrange(emitted, items)
	if len(r.Entries) > 0 {
		file.Decls = append(file.Decls, &ast.TargetDecl{DeclHead: ast.DeclHead{N: Name}, For: pkg, Entries: r.Entries})
	}
	return reverse.Lower(file)
}

func (r *reader) pos(p *gqlast.Position) *ir.Position {
	if p == nil {
		return &ir.Position{Filename: r.path}
	}
	return &ir.Position{Filename: r.path, Line: int32(p.Line), Column: int32(p.Column)}
}

func (r *reader) at(p *gqlast.Position) ast.Position {
	if p == nil {
		return ast.Position{Filename: r.path}
	}
	return ast.Position{Filename: r.path, Line: p.Line, Col: p.Column}
}

// header reads what the schema says of the whole file: the package, from
// the file's name, and under roundtrip the schema extension's annotations.
// What else a schema definition, an extension, or a directive definition
// says has no TDL form.
func (r *reader) header(file *ast.File) (string, []reverse.Item, []string, error) {
	pkg := strings.TrimSuffix(path.Base(r.path), path.Ext(r.path))
	switch {
	case pkg == defaultFile:
		pkg = ""
	case !ident.MatchString(pkg):
		r.Warn(emit.LossName, r.pos(nil), "%s names no package, and regenerates as %s.graphql", r.path, defaultFile)
		pkg = ""
	}

	var anns []kv
	for _, s := range slices.Concat(r.doc.Schema, r.doc.SchemaExtension) {
		anns = append(anns, tdl(s.Directives)...)
		r.unknown(s.Directives, s.Position, "the schema")
		if len(s.OperationTypes) > 0 && !r.Roundtrip {
			r.Warn(emit.LossUnsupported, r.pos(s.Position), "the schema's root operation types have no TDL form")
		}
	}
	for _, d := range r.doc.Directives {
		if d.Name != directive {
			r.Warn(emit.LossUnsupported, r.pos(d.Position), "directive @%s has no TDL form", d.Name)
		}
	}
	for _, ext := range r.doc.Extensions {
		r.Warn(emit.LossUnsupported, r.pos(ext.Position), "the extension of %s is not read", ext.Name)
	}

	if v, ok := get(anns, "package"); ok {
		pkg = v
	}
	var doc []string
	if v, ok := get(anns, "doc"); ok {
		doc = strings.Split(v, "\n")
	}
	for _, src := range all(anns, "import") {
		imp, err := reverse.ParseImport(src)
		if err != nil {
			return "", nil, nil, reverse.Failf(r.pos(nil), "@tdl(import: %q) is not one TDL import: %v", src, err)
		}
		file.Imports = append(file.Imports, imp)
	}
	var items []reverse.Item
	for _, v := range all(anns, "item") {
		var at int
		if n, err := fmt.Sscanf(v, "%d", &at); n != 1 || err != nil {
			return "", nil, nil, reverse.Failf(r.pos(nil), "a @tdl(item:) does not start with its index")
		}
		_, src, _ := strings.Cut(v, " ")
		decl, err := reverse.ParseItem(src)
		if err != nil {
			return "", nil, nil, reverse.Failf(r.pos(nil), "a @tdl(item:) is not TDL: %v", err)
		}
		items = append(items, reverse.Item{Decl: decl, At: at})
	}
	return pkg, items, doc, nil
}

// nameAll gives every type import reads its TDL name before any is read,
// since a field may name one defined after it, and finds the unions
// written as fielded enums.
func (r *reader) nameAll() {
	objects := map[string]*gqlast.Definition{}
	for _, def := range r.doc.Definitions {
		switch def.Kind {
		case gqlast.Object:
			objects[def.Name] = def
			r.names[def.Name] = declName(def.Directives, def.Name)
		case gqlast.Enum:
			r.names[def.Name] = declName(def.Directives, def.Name)
		}
	}

	uses := map[string]int{}
	for _, def := range r.doc.Definitions {
		for _, f := range def.Fields {
			uses[f.Type.Name()]++
			for _, a := range f.Arguments {
				uses[a.Type.Name()]++
			}
		}
		for _, m := range def.Types {
			uses[m]++
		}
	}
	for _, def := range r.doc.Definitions {
		if def.Kind != gqlast.Union {
			continue
		}
		sum := len(def.Types) > 0
		for _, m := range def.Types {
			o := objects[m]
			sum = sum && o != nil && uses[m] == 1 && len(o.Interfaces) == 0 && !r.variants[m]
		}
		if !sum {
			continue
		}
		r.sums[def.Name] = true
		r.names[def.Name] = declName(def.Directives, def.Name)
		for _, m := range def.Types {
			r.variants[m] = true
			delete(r.names, m)
		}
	}
}

// declName is a type's TDL name: its annotation's, or its own.
func declName(dirs gqlast.DirectiveList, name string) string {
	if v, ok := get(tdl(dirs), "name"); ok {
		return v
	}
	return name
}

// definition reads one type, or returns nil for one with no declaration
// of its own.
func (r *reader) definition(def *gqlast.Definition) (ast.Decl, error) {
	switch def.Kind {
	case gqlast.Object:
		if r.variants[def.Name] {
			return nil, nil
		}
		return r.structure(def)
	case gqlast.Enum:
		return r.enum(def)
	case gqlast.Union:
		return r.union(def)
	case gqlast.Scalar:
		r.scalar(def)
		return nil, nil
	}
	r.Warn(emit.LossUnsupported, r.pos(def.Position), "%s %s has no TDL form", strings.ToLower(strings.ReplaceAll(string(def.Kind), "_", " ")), def.Name)
	return nil, nil
}

// scalar checks a scalar declaration, which declares nothing: a known one
// regenerates where it is used.
func (r *reader) scalar(def *gqlast.Definition) {
	if _, ok := primitives[def.Name]; !ok {
		r.Warn(emit.LossUnsupported, r.pos(def.Position), "scalar %s has no TDL form", def.Name)
		return
	}
	if !r.Roundtrip && def.Description != "" && def.Description != custom[def.Name] {
		r.Warn(emit.LossDoc, r.pos(def.Position), "scalar %s's description is not read", def.Name)
	}
	r.unknown(def.Directives, def.Position, def.Name)
}

// head is a node's head: its doc comment and deprecation from its
// description and @deprecated, or from its annotations.
func (r *reader) head(name, desc string, dirs gqlast.DirectiveList, p *gqlast.Position, kind node) ast.DeclHead {
	anns := tdl(dirs)
	dep := dirs.ForName("deprecated")
	var doc []string
	var deprecation *ast.Deprecation
	if v, ok := get(anns, "doc"); ok {
		if v != "" {
			doc = strings.Split(v, "\n")
		}
		if reason, ok := get(anns, "reason"); ok {
			deprecation = &ast.Deprecation{Reason: reason}
		}
	} else {
		doc, deprecation = readDescription(desc, kind)
	}
	if dep != nil && deprecation == nil {
		deprecation = &ast.Deprecation{}
		if a := dep.Arguments.ForName("reason"); a != nil && a.Value != nil {
			deprecation.Reason = a.Value.Raw
		}
	}
	h := reverse.Head(name, doc, r.at(p), false)
	h.Dep = deprecation
	return h
}

// unknown warns about directives with no TDL form, unless roundtrip.
func (r *reader) unknown(dirs gqlast.DirectiveList, p *gqlast.Position, what string) {
	if r.Roundtrip {
		return
	}
	for _, d := range dirs {
		if d.Name != "deprecated" && d.Name != directive {
			r.Warn(emit.LossUnsupported, r.pos(cmp.Or(d.Position, p)), "%s's directive @%s has no TDL form", what, d.Name)
		}
	}
}

// rename writes a name directive on a type whose GraphQL name the
// generator would not derive from its TDL name.
func (r *reader) rename(name, written string) {
	if !r.Roundtrip && emit.Pascal(written) != written {
		r.Directive(name, "name", reverse.StrLit(written))
	}
}

func (r *reader) structure(def *gqlast.Definition) (ast.Decl, error) {
	name := r.names[def.Name]
	decl := &ast.StructDecl{DeclHead: r.head(name, def.Description, def.Directives, def.Position, object), Keyword: "type"}
	anns := tdl(def.Directives)
	switch v, _ := get(anns, "kind"); v {
	case "entity":
		decl.Conforms = []*ast.ClassRef{{N: "Entity"}}
	case "mixin":
		decl.Keyword = "mixin"
	}
	conforms, err := r.conforms(anns, def)
	if err != nil {
		return nil, err
	}
	if conforms != nil {
		decl.Conforms = conforms
	}
	if len(def.Interfaces) > 0 {
		r.Warn(emit.LossUnsupported, r.pos(def.Position), "%s implements %s, which TDL has no form for", def.Name, strings.Join(def.Interfaces, ", "))
	}
	r.rename(name, def.Name)
	r.unknown(def.Directives, def.Position, def.Name)

	fields, err := r.fields(def.Fields, def.Name)
	if err != nil {
		return nil, err
	}
	include := ""
	for _, f := range fields {
		inc, _ := get(f.anns, "include")
		if inc == "" {
			include = ""
			decl.Members = append(decl.Members, f.field)
			continue
		}
		if inc != include {
			include = inc
			decl.Members = append(decl.Members, &ast.Include{P: f.field.P, Type: &ast.ClassRef{N: inc}})
		}
	}
	if len(decl.Members) > 0 {
		decl.End = decl.P
	}
	return decl, nil
}

// conforms is a conforms annotation's conformance list, or nil.
func (r *reader) conforms(anns []kv, def *gqlast.Definition) ([]*ast.ClassRef, error) {
	src, ok := get(anns, "conforms")
	if !ok {
		return nil, nil
	}
	out, err := reverse.ParseConforms(src)
	if err != nil {
		return nil, reverse.Failf(r.pos(def.Position), "the conforms of %s is not a TDL conformance list: %v", def.Name, err)
	}
	return out, nil
}

// field is a field read, with its annotations.
type field struct {
	field *ast.Field
	anns  []kv
}

// fields reads an object's fields, skipping one with no TDL form.
func (r *reader) fields(in gqlast.FieldList, owner string) ([]field, error) {
	var out []field
	for _, f := range in {
		af, err := r.field(f, owner)
		if err != nil {
			return nil, err
		}
		if af != nil {
			out = append(out, field{af, tdl(f.Directives)})
		}
	}
	return out, nil
}

// field reads one field: from its annotation under roundtrip, and
// otherwise from its name and type. It returns nil for a field it skips.
func (r *reader) field(f *gqlast.FieldDefinition, owner string) (*ast.Field, error) {
	var out *ast.Field
	if src, ok := get(tdl(f.Directives), "source"); ok {
		parsed, err := reverse.ParseField(src)
		if err != nil {
			return nil, reverse.Failf(r.pos(f.Position), "the source of %s.%s is not a TDL field: %v", owner, f.Name, err)
		}
		out = parsed
	} else {
		typ := r.typeRef(f.Type, f.Position, owner+"."+f.Name)
		if typ == nil {
			return nil, nil
		}
		out = &ast.Field{Type: typ}
		out.N = f.Name
	}
	if len(f.Arguments) > 0 {
		r.Warn(emit.LossUnsupported, r.pos(f.Position), "%s.%s takes arguments, which are not read", owner, f.Name)
	}
	out.DeclHead = r.head(out.N, f.Description, f.Directives, f.Position, member)
	r.unknown(f.Directives, f.Position, owner+"."+f.Name)
	return out, nil
}

func (r *reader) enum(def *gqlast.Definition) (ast.Decl, error) {
	name := r.names[def.Name]
	decl := &ast.EnumDecl{DeclHead: r.head(name, def.Description, def.Directives, def.Position, object)}
	conforms, err := r.conforms(tdl(def.Directives), def)
	if err != nil {
		return nil, err
	}
	decl.Conforms = conforms
	r.rename(name, def.Name)
	r.unknown(def.Directives, def.Position, def.Name)

	for _, v := range def.EnumValues {
		vname := emit.FromScreaming(v.Name)
		if n, ok := get(tdl(v.Directives), "name"); ok {
			vname = n
		} else if !r.Roundtrip && emit.ScreamingSnake(vname) != v.Name {
			r.Directive(name+"."+vname, "name", reverse.StrLit(v.Name))
		}
		decl.Variants = append(decl.Variants, &ast.Variant{DeclHead: r.head(vname, v.Description, v.Directives, v.Position, member)})
		r.unknown(v.Directives, v.Position, def.Name+"."+v.Name)
	}
	return decl, nil
}

// union reads a union written as a fielded enum, and skips any other.
func (r *reader) union(def *gqlast.Definition) (ast.Decl, error) {
	if !r.sums[def.Name] {
		r.Warn(emit.LossUnsupported, r.pos(def.Position), "union %s holds something other than object types used nowhere else, which TDL has no form for", def.Name)
		return nil, nil
	}
	name := r.names[def.Name]
	decl := &ast.EnumDecl{DeclHead: r.head(name, def.Description, def.Directives, def.Position, object)}
	conforms, err := r.conforms(tdl(def.Directives), def)
	if err != nil {
		return nil, err
	}
	decl.Conforms = conforms
	r.rename(name, def.Name)
	r.unknown(def.Directives, def.Position, def.Name)

	for _, m := range def.Types {
		o := r.doc.Definitions.ForName(m)
		vname := variantBack(def.Name, m)
		if n, ok := get(tdl(o.Directives), "name"); ok {
			vname = n
		}
		path := name + "." + vname
		if !r.Roundtrip && m != def.Name+emit.Pascal(vname) {
			r.Directive(path, "name", reverse.StrLit(m))
		}
		kind := object
		if fieldless(o) {
			kind = empty
		}
		v := &ast.Variant{DeclHead: r.head(vname, o.Description, o.Directives, o.Position, kind)}
		r.unknown(o.Directives, o.Position, m)
		if kind == object {
			fields, err := r.fields(o.Fields, path)
			if err != nil {
				return nil, err
			}
			for _, f := range fields {
				v.Fields = append(v.Fields, f.field)
			}
			if len(v.Fields) > 0 {
				v.End = v.P
			}
		}
		decl.Variants = append(decl.Variants, v)
	}
	return decl, nil
}

// fieldless reports whether an object type is a fieldless variant's: one
// nullable Boolean named for the placeholder.
func fieldless(o *gqlast.Definition) bool {
	if len(o.Fields) != 1 {
		return false
	}
	f := o.Fields[0]
	return f.Name == placeholder && f.Type.NamedType == "Boolean" && !f.Type.NonNull && len(f.Arguments) == 0
}

// typeRef reads a type, or warns and returns nil.
func (r *reader) typeRef(t *gqlast.Type, p *gqlast.Position, what string) *ast.TypeRef {
	var out *ast.TypeRef
	switch {
	case t.Elem != nil:
		elem := r.typeRef(t.Elem, p, what)
		if elem == nil {
			return nil
		}
		out = &ast.TypeRef{List: elem}
	case primitives[t.NamedType] != "":
		if t.NamedType == "ID" {
			r.Warn(emit.LossPrimitive, r.pos(p), "%s is an ID, which regenerates as String", what)
		}
		out = &ast.TypeRef{N: primitives[t.NamedType]}
	case r.names[t.NamedType] != "":
		out = &ast.TypeRef{N: r.names[t.NamedType]}
	default:
		r.Warn(emit.LossUnsupported, r.pos(p), "%s names %s, which is not imported", what, t.NamedType)
		return nil
	}
	out.Optional = !t.NonNull
	return out
}

// tdl is the arguments of every @tdl directive in a list, in order.
func tdl(dirs gqlast.DirectiveList) []kv {
	var out []kv
	for _, d := range dirs {
		if d.Name != directive {
			continue
		}
		for _, a := range d.Arguments {
			if a.Value != nil {
				out = append(out, kv{a.Name, a.Value.Raw})
			}
		}
	}
	return out
}

// get is an annotation's value, and whether it is present.
func get(anns []kv, key string) (string, bool) {
	for _, a := range anns {
		if a.key == key {
			return a.value, true
		}
	}
	return "", false
}

// all is every value of a repeated annotation.
func all(anns []kv, key string) []string {
	var out []string
	for _, a := range anns {
		if a.key == key {
			out = append(out, a.value)
		}
	}
	return out
}

// Normalize puts .graphql files into the normal form the round-trip corpus
// compares: the schema gqlparser parses, formatted without comments, each
// kind of definition sorted by name, and scalars without descriptions,
// which no model carries.
func Normalize(files []*plugin.File) ([]*plugin.File, error) {
	out := make([]*plugin.File, len(files))
	for i, f := range files {
		doc, err := gqlparse.ParseSchema(&gqlast.Source{Name: f.GetPath(), Input: string(f.GetContent())})
		if err != nil {
			return nil, fmt.Errorf("%s: %w", f.GetPath(), err)
		}
		for _, def := range doc.Definitions {
			if def.Kind == gqlast.Scalar {
				def.Description = ""
			}
		}
		slices.SortStableFunc(doc.Definitions, func(a, b *gqlast.Definition) int { return cmp.Compare(a.Name, b.Name) })
		slices.SortStableFunc(doc.Directives, func(a, b *gqlast.DirectiveDefinition) int { return cmp.Compare(a.Name, b.Name) })
		var b bytes.Buffer
		formatter.NewFormatter(&b).FormatSchemaDocument(doc)
		out[i] = &plugin.File{Path: f.GetPath(), Content: b.Bytes()}
	}
	return out, nil
}
