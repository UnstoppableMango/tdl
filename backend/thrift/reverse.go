package thrift

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/cloudwego/thriftgo/parser"

	"github.com/unstoppablemango/tdl/ast"
	"github.com/unstoppablemango/tdl/backend/internal/emit"
	"github.com/unstoppablemango/tdl/backend/internal/reverse"
	"github.com/unstoppablemango/tdl/ir"
	"github.com/unstoppablemango/tdl/plugin"
)

// Import reads one .thrift file into a model: each struct a struct, each
// enum an enum, each typedef a newtype, and a union of structs nothing
// else names, as Generate writes a fielded enum, an enum with fields.
// What regenerating needs goes in a thrift target block. Under the tdl.*
// annotations a `roundtrip` directive writes, it rebuilds the model the
// file was generated from. docs/design/schema-backends.md has the mapping.
func (Backend) Import(_ context.Context, req *plugin.ImportRequest) (*plugin.ImportResponse, error) {
	r := &reader{names: map[string]string{}, sums: map[string]bool{}, variants: map[string]bool{}}
	return r.Response(r.read(req.GetFiles()))
}

type reader struct {
	reverse.Reader
	path string
	t    *parser.Thrift

	// at maps each definition to where it starts.
	at map[any]ast.Position

	// names maps each Thrift type the file declares and import reads to
	// its TDL name.
	names map[string]string
	// sums is the unions read as enums, and variants the structs they
	// absorb.
	sums     map[string]bool
	variants map[string]bool
}

func (r *reader) read(files []*plugin.File) (*ir.Model, error) {
	if len(files) != 1 {
		return nil, reverse.Failf(nil, "import reads one .thrift file, and was given %d", len(files))
	}
	r.path = files[0].GetPath()
	src := string(files[0].GetContent())
	t, err := parser.ParseString(r.path, src)
	if err != nil {
		return nil, reverse.Failf(r.pos(ast.Position{}), "%s does not parse: %v", r.path, err)
	}
	r.t = t
	if len(t.Includes) > 0 {
		return nil, reverse.Failf(r.pos(ast.Position{}), "%s includes %s, and includes are not imported yet", r.path, t.Includes[0].Path)
	}
	r.at = r.locate(src)
	r.Roundtrip = annotated(t)

	file := &ast.File{Filename: r.path}
	pkg, items, doc, err := r.header(file)
	if err != nil {
		return nil, err
	}
	if pkg != "" || len(doc) > 0 {
		file.Package = &ast.PackageDecl{Path: pkg, Doc: doc, DocP: make([]ast.Position, len(doc))}
	}

	r.nameAll()
	emitted, err := r.definitions()
	if err != nil {
		return nil, err
	}
	file.Decls = reverse.Arrange(emitted, items)
	if len(r.Entries) > 0 {
		file.Decls = append(file.Decls, &ast.TargetDecl{DeclHead: ast.DeclHead{N: Name}, For: pkg, Entries: r.Entries})
	}
	return reverse.Lower(file)
}

func (r *reader) pos(p ast.Position) *ir.Position {
	return &ir.Position{Filename: r.path, Line: int32(p.Line), Column: int32(p.Col)}
}

// annotated reports whether a file carries any tdl.* annotation, which
// only a `roundtrip` directive writes.
func annotated(t *parser.Thrift) bool {
	var all []parser.Annotations
	for _, ns := range t.Namespaces {
		all = append(all, ns.Annotations)
	}
	for _, td := range t.Typedefs {
		all = append(all, td.Annotations)
	}
	for _, e := range t.Enums {
		all = append(all, e.Annotations)
		for _, v := range e.Values {
			all = append(all, v.Annotations)
		}
	}
	for _, s := range slices.Concat(t.Structs, t.Unions, t.Exceptions) {
		all = append(all, s.Annotations)
		for _, f := range s.Fields {
			all = append(all, f.Annotations)
		}
	}
	for _, a := range all {
		for _, kv := range a {
			if strings.HasPrefix(kv.Key, "tdl.") {
				return true
			}
		}
	}
	return false
}

// header reads the namespaces: the package, and under roundtrip the
// file's annotations.
func (r *reader) header(file *ast.File) (string, []reverse.Item, []string, error) {
	pkg := ""
	var anns parser.Annotations
	for _, ns := range r.t.Namespaces {
		switch ns.Language {
		case "*":
			pkg = ns.Name
		case carrier:
		default:
			if !r.Roundtrip {
				r.Warn(emit.LossUnsupported, r.pos(ast.Position{}), "namespace %s %s regenerates as namespace *", ns.Language, ns.Name)
			}
		}
		anns = append(anns, ns.Annotations...)
	}
	if v, ok := get(anns, "tdl.package"); ok {
		pkg = v
	}
	var doc []string
	if v, ok := get(anns, "tdl.doc"); ok {
		doc = strings.Split(v, "\n")
	}
	for _, src := range all(anns, "tdl.import") {
		imp, err := reverse.ParseImport(src)
		if err != nil {
			return "", nil, nil, reverse.Failf(r.pos(ast.Position{}), "tdl.import %q is not one TDL import: %v", src, err)
		}
		file.Imports = append(file.Imports, imp)
	}
	var items []reverse.Item
	for _, v := range all(anns, "tdl.item") {
		var at int
		var src string
		if n, err := fmt.Sscanf(v, "%d", &at); n != 1 || err != nil {
			return "", nil, nil, reverse.Failf(r.pos(ast.Position{}), "a tdl.item does not start with its index")
		}
		_, src, _ = strings.Cut(v, " ")
		decl, err := reverse.ParseItem(src)
		if err != nil {
			return "", nil, nil, reverse.Failf(r.pos(ast.Position{}), "a tdl.item is not TDL: %v", err)
		}
		items = append(items, reverse.Item{Decl: decl, At: at})
	}
	return pkg, items, doc, nil
}

// nameAll gives every definition import reads its TDL name before any is
// read, since a field may name one defined after it, and finds the unions
// written as fielded enums.
func (r *reader) nameAll() {
	for _, td := range r.t.Typedefs {
		r.names[td.Alias] = declName(td.Annotations, td.Alias)
	}
	for _, e := range r.t.Enums {
		r.names[e.Name] = declName(e.Annotations, e.Name)
	}
	for _, s := range slices.Concat(r.t.Structs, r.t.Exceptions) {
		r.names[s.Name] = declName(s.Annotations, s.Name)
	}

	uses := map[string]int{}
	var count func(t *parser.Type)
	count = func(t *parser.Type) {
		if t == nil {
			return
		}
		uses[t.Name]++
		count(t.KeyType)
		count(t.ValueType)
	}
	for _, td := range r.t.Typedefs {
		count(td.Type)
	}
	for _, c := range r.t.Constants {
		count(c.Type)
	}
	for _, s := range slices.Concat(r.t.Structs, r.t.Unions, r.t.Exceptions) {
		for _, f := range s.Fields {
			count(f.Type)
		}
	}
	for _, svc := range r.t.Services {
		for _, fn := range svc.Functions {
			count(fn.FunctionType)
			for _, f := range slices.Concat(fn.Arguments, fn.Throws) {
				count(f.Type)
			}
		}
	}
	structs := map[string]bool{}
	for _, s := range r.t.Structs {
		structs[s.Name] = true
	}
	for _, u := range r.t.Unions {
		sum := len(u.Fields) > 0
		for _, f := range u.Fields {
			sum = sum && structs[f.Type.Name] && uses[f.Type.Name] == 1 && f.Default == nil
		}
		if !sum {
			continue
		}
		r.sums[u.Name] = true
		r.names[u.Name] = declName(u.Annotations, u.Name)
		for _, f := range u.Fields {
			r.variants[f.Type.Name] = true
			delete(r.names, f.Type.Name)
		}
	}
}

// declName is a definition's TDL name: its annotation's, or its own.
func declName(anns parser.Annotations, name string) string {
	if v, ok := get(anns, "tdl.name"); ok {
		return v
	}
	return name
}

// definitions reads every definition in source order.
func (r *reader) definitions() ([]reverse.Item, error) {
	type def struct {
		at   ast.Position
		read func() (*reverse.Item, error)
	}
	var defs []def
	for _, td := range r.t.Typedefs {
		defs = append(defs, def{r.at[td], func() (*reverse.Item, error) { return r.typedef(td) }})
	}
	for _, e := range r.t.Enums {
		defs = append(defs, def{r.at[e], func() (*reverse.Item, error) { return r.enum(e) }})
	}
	for _, s := range slices.Concat(r.t.Structs, r.t.Exceptions) {
		if r.variants[s.Name] {
			continue
		}
		defs = append(defs, def{r.at[s], func() (*reverse.Item, error) { return r.structure(s) }})
	}
	for _, u := range r.t.Unions {
		defs = append(defs, def{r.at[u], func() (*reverse.Item, error) { return r.union(u) }})
	}
	for _, c := range r.t.Constants {
		r.Warn(emit.LossUnsupported, r.pos(r.at[c]), "constant %s has no TDL form", c.Name)
	}
	for _, svc := range r.t.Services {
		r.Warn(emit.LossUnsupported, r.pos(r.at[svc]), "service %s is not imported yet", svc.Name)
	}
	slices.SortStableFunc(defs, func(a, b def) int {
		return cmp.Or(cmp.Compare(a.at.Line, b.at.Line), cmp.Compare(a.at.Col, b.at.Col))
	})

	var out []reverse.Item
	for _, d := range defs {
		it, err := d.read()
		if err != nil {
			return nil, err
		}
		if it != nil {
			out = append(out, *it)
		}
	}
	return out, nil
}

// head is a node's head from its comment, deprecation, and annotations.
func (r *reader) head(name, comments string, anns parser.Annotations, pos ast.Position) ast.DeclHead {
	_, deprecated := get(anns, "deprecated")
	doc := docLines(comments)
	if v, ok := get(anns, "tdl.doc"); ok {
		doc = strings.Split(v, "\n")
		if v == "" {
			doc = nil
		}
		h := reverse.Head(name, doc, pos, false)
		if reason, ok := get(anns, "tdl.reason"); ok && deprecated {
			h.Dep = &ast.Deprecation{Reason: reason}
		}
		return h
	}
	h := reverse.Head(name, doc, pos, deprecated)
	if deprecated && h.Dep.Reason == "" && len(doc) == 0 {
		// A handwritten file says why in the annotation alone.
		h.Dep.Reason, _ = get(anns, "deprecated")
	}
	return h
}

// at is the index an annotation gives a declaration, or -1.
func at(anns parser.Annotations) int {
	v, ok := get(anns, "tdl.at")
	if !ok {
		return -1
	}
	var n int
	if _, err := fmt.Sscanf(v, "%d", &n); err != nil {
		return -1
	}
	return n
}

// unknown warns about annotations with no TDL form, unless roundtrip.
func (r *reader) unknown(anns parser.Annotations, pos ast.Position, what string) {
	if r.Roundtrip {
		return
	}
	for _, a := range anns {
		if a.Key != "deprecated" && !strings.HasPrefix(a.Key, "tdl.") {
			r.Warn(emit.LossUnsupported, r.pos(pos), "%s's annotation %s has no TDL form", what, a.Key)
		}
	}
}

func (r *reader) typedef(td *parser.Typedef) (*reverse.Item, error) {
	pos := r.at[td]
	name := r.names[td.Alias]
	var decl *ast.NewtypeDecl
	if src, ok := get(td.Annotations, "tdl.source"); ok {
		parsed, err := reverse.ParseItem("type " + src)
		n, isNewtype := parsed.(*ast.NewtypeDecl)
		if err != nil || !isNewtype {
			return nil, reverse.Failf(r.pos(pos), "the tdl.source of %s is not a TDL newtype: %v", td.Alias, err)
		}
		decl = n
	} else {
		base := r.typeRef(td.Type, pos, td.Alias)
		if base == nil {
			return nil, nil
		}
		decl = &ast.NewtypeDecl{Base: base}
		decl.N = name
		r.rename(name, td.Alias)
	}
	h := r.head(decl.N, td.ReservedComments, td.Annotations, pos)
	decl.DeclHead = h
	r.unknown(td.Annotations, pos, td.Alias)
	return &reverse.Item{Decl: decl, At: at(td.Annotations)}, nil
}

// rename writes a name directive on a declaration whose Thrift name the
// generator would not derive from its TDL name.
func (r *reader) rename(name, written string) {
	if emit.Pascal(written) != written {
		r.Directive(name, "name", reverse.StrLit(written))
	}
}

func (r *reader) structure(s *parser.StructLike) (*reverse.Item, error) {
	pos := r.at[s]
	name := r.names[s.Name]
	decl := &ast.StructDecl{DeclHead: r.head(name, s.ReservedComments, s.Annotations, pos), Keyword: "type"}
	switch v, _ := get(s.Annotations, "tdl.kind"); v {
	case "entity":
		decl.Conforms = []*ast.ClassRef{{N: "Entity"}}
	case "mixin":
		decl.Keyword = "mixin"
	}
	conforms, err := r.conforms(s.Annotations, pos, s.Name)
	if err != nil {
		return nil, err
	}
	if conforms != nil {
		decl.Conforms = conforms
	}
	if s.Category == "exception" {
		r.Warn(emit.LossUnsupported, r.pos(pos), "exception %s is read as a struct", s.Name)
	}
	if !r.Roundtrip {
		r.rename(name, s.Name)
	}
	r.unknown(s.Annotations, pos, s.Name)

	fields, err := r.fields(s.Fields, pos, name)
	if err != nil {
		return nil, err
	}
	include := ""
	for _, f := range fields {
		inc, _ := get(f.anns, "tdl.include")
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
		decl.End = pos
	}
	return &reverse.Item{Decl: decl, At: at(s.Annotations)}, nil
}

// conforms is a tdl.conforms annotation's conformance list, or nil.
func (r *reader) conforms(anns parser.Annotations, pos ast.Position, name string) ([]*ast.ClassRef, error) {
	src, ok := get(anns, "tdl.conforms")
	if !ok {
		return nil, nil
	}
	out, err := reverse.ParseConforms(src)
	if err != nil {
		return nil, reverse.Failf(r.pos(pos), "the tdl.conforms of %s is not a TDL conformance list: %v", name, err)
	}
	return out, nil
}

// field is a field read, with its annotations.
type field struct {
	field *ast.Field
	anns  parser.Annotations
}

// fields reads a struct's fields, pinning the ids allocation would not
// give, and skips one with no TDL form.
func (r *reader) fields(in []*parser.Field, pos ast.Position, owner string) ([]field, error) {
	var out []field
	var names []string
	var ids []int64
	numbered := true
	for _, f := range in {
		af, err := r.field(f, pos, owner)
		if err != nil {
			return nil, err
		}
		if af == nil {
			continue
		}
		out = append(out, field{af, f.Annotations})
		names = append(names, owner+"."+af.N)
		ids = append(ids, int64(f.ID))
		if f.ID < 1 {
			numbered = false
		}
	}
	if !numbered {
		r.Warn(emit.LossNumber, r.pos(pos), "%s has a field without a positive id, and its fields regenerate numbered from 1", owner)
		return out, nil
	}
	r.Pins(names, ids, fieldNumbers)
	return out, nil
}

// field reads one field: from its annotation under roundtrip, and
// otherwise from its name and type. It returns nil for a field it skips.
func (r *reader) field(f *parser.Field, pos ast.Position, owner string) (*ast.Field, error) {
	var out *ast.Field
	if src, ok := get(f.Annotations, "tdl.source"); ok {
		parsed, err := reverse.ParseField(src)
		if err != nil {
			return nil, reverse.Failf(r.pos(pos), "the tdl.source of %s.%s is not a TDL field: %v", owner, f.Name, err)
		}
		out = parsed
	} else {
		typ := r.typeRef(f.Type, pos, owner+"."+f.Name)
		if typ == nil {
			return nil, nil
		}
		switch f.Requiredness {
		case parser.FieldType_Optional:
			typ.Optional = true
		case parser.FieldType_Required:
			r.Warn(emit.LossUnsupported, r.pos(pos), "%s.%s is required, which regenerates with default requiredness", owner, f.Name)
		}
		if f.Default != nil {
			r.Warn(emit.LossDefault, r.pos(pos), "%s.%s has a default, which is not read", owner, f.Name)
		}
		out = &ast.Field{Type: typ}
		out.N = f.Name
	}
	out.DeclHead = r.head(out.N, f.ReservedComments, f.Annotations, pos)
	r.unknown(f.Annotations, pos, owner+"."+f.Name)
	return out, nil
}

func (r *reader) enum(e *parser.Enum) (*reverse.Item, error) {
	pos := r.at[e]
	name := r.names[e.Name]
	decl := &ast.EnumDecl{DeclHead: r.head(name, e.ReservedComments, e.Annotations, pos)}
	conforms, err := r.conforms(e.Annotations, pos, e.Name)
	if err != nil {
		return nil, err
	}
	decl.Conforms = conforms
	if !r.Roundtrip {
		r.rename(name, e.Name)
	}
	r.unknown(e.Annotations, pos, e.Name)

	var names []string
	var nums []int64
	numbered := true
	for _, v := range e.Values {
		vname := emit.FromScreaming(v.Name)
		if n, ok := get(v.Annotations, "tdl.name"); ok {
			vname = n
		} else if !r.Roundtrip && emit.ScreamingSnake(vname) != v.Name {
			r.Directive(name+"."+vname, "name", reverse.StrLit(v.Name))
		}
		decl.Variants = append(decl.Variants, &ast.Variant{DeclHead: r.head(vname, v.ReservedComments, v.Annotations, pos)})
		r.unknown(v.Annotations, pos, e.Name+"."+v.Name)
		names = append(names, name+"."+vname)
		nums = append(nums, v.Value)
		if v.Value < 1 {
			numbered = false
		}
	}
	if !numbered {
		r.Warn(emit.LossNumber, r.pos(pos), "%s has a value below 1, and its values regenerate numbered from 1", e.Name)
	} else {
		r.Pins(names, nums, enumNumbers)
	}
	return &reverse.Item{Decl: decl, At: at(e.Annotations)}, nil
}

// union reads a union written as a fielded enum, and skips any other.
func (r *reader) union(u *parser.StructLike) (*reverse.Item, error) {
	pos := r.at[u]
	if !r.sums[u.Name] {
		r.Warn(emit.LossUnsupported, r.pos(pos), "union %s holds something other than structs used nowhere else, which TDL has no form for", u.Name)
		return nil, nil
	}
	name := r.names[u.Name]
	decl := &ast.EnumDecl{DeclHead: r.head(name, u.ReservedComments, u.Annotations, pos)}
	conforms, err := r.conforms(u.Annotations, pos, u.Name)
	if err != nil {
		return nil, err
	}
	decl.Conforms = conforms
	if !r.Roundtrip {
		r.rename(name, u.Name)
	}
	r.unknown(u.Annotations, pos, u.Name)

	var names []string
	var nums []int64
	for _, m := range u.Fields {
		s := r.structNamed(m.Type.Name)
		spos := r.at[s]
		vname := variantBack(u.Name, s.Name, m.Name)
		if n, ok := get(m.Annotations, "tdl.name"); ok {
			vname = n
		} else if !r.Roundtrip && emit.Camel(vname) != m.Name {
			r.Warn(emit.LossName, r.pos(pos), "%s's member %s regenerates as %s", u.Name, m.Name, emit.Camel(vname))
		}
		path := name + "." + vname
		if !r.Roundtrip && s.Name != u.Name+vname {
			r.Directive(path, "name", reverse.StrLit(s.Name))
		}
		v := &ast.Variant{DeclHead: r.head(vname, s.ReservedComments, s.Annotations, spos)}
		r.unknown(s.Annotations, spos, s.Name)
		fields, err := r.fields(s.Fields, spos, path)
		if err != nil {
			return nil, err
		}
		for _, f := range fields {
			v.Fields = append(v.Fields, f.field)
		}
		if len(v.Fields) > 0 {
			v.End = v.P
		}
		decl.Variants = append(decl.Variants, v)
		names = append(names, path)
		nums = append(nums, int64(m.ID))
	}
	r.Pins(names, nums, fieldNumbers)
	return &reverse.Item{Decl: decl, At: at(u.Annotations)}, nil
}

// variantBack is the variant a union member reads back as: what its
// struct's name adds to the union's when the member is that in camel case,
// and otherwise the member in Pascal case.
func variantBack(union, structName, member string) string {
	if rest, ok := strings.CutPrefix(structName, union); ok && rest != "" && emit.Camel(rest) == member {
		return rest
	}
	return emit.Pascal(member)
}

func (r *reader) structNamed(name string) *parser.StructLike {
	for _, s := range r.t.Structs {
		if s.Name == name {
			return s
		}
	}
	return nil
}

// bases maps a Thrift base type to the TDL primitive it reads as.
var bases = map[string]string{
	"bool":   "bool",
	"i32":    "int32",
	"i64":    "int64",
	"double": "float64",
	"string": "string",
	"binary": "bytes",
}

// narrow is the base types TDL has no primitive for, read as int32.
var narrow = map[string]bool{"byte": true, "i8": true, "i16": true}

// typeRef reads a type, or warns and returns nil.
func (r *reader) typeRef(t *parser.Type, pos ast.Position, what string) *ast.TypeRef {
	switch t.Name {
	case "list", "set":
		elem := r.typeRef(t.ValueType, pos, what)
		if elem == nil {
			return nil
		}
		if t.Name == "set" {
			return &ast.TypeRef{Set: elem}
		}
		return &ast.TypeRef{List: elem}
	case "map":
		k, v := r.typeRef(t.KeyType, pos, what), r.typeRef(t.ValueType, pos, what)
		if k == nil || v == nil {
			return nil
		}
		return &ast.TypeRef{MapKey: k, MapValue: v}
	}
	if p, ok := bases[t.Name]; ok {
		return &ast.TypeRef{N: p}
	}
	if narrow[t.Name] {
		r.Warn(emit.LossPrimitive, r.pos(pos), "%s is %s, which regenerates as i32", what, t.Name)
		return &ast.TypeRef{N: "int32"}
	}
	if n, ok := r.names[t.Name]; ok {
		return &ast.TypeRef{N: n}
	}
	r.Warn(emit.LossUnsupported, r.pos(pos), "%s names %s, which is not imported", what, t.Name)
	return nil
}

// locate finds where each definition starts. thriftgo keeps no
// positions, so the source is scanned for the keywords that open one, at
// the top level, outside comments and strings; the nth of each keyword is
// the nth definition of its kind.
func (r *reader) locate(src string) map[any]ast.Position {
	kinds := map[string][]ast.Position{}
	line, col := 1, 1
	depth := 0
	advance := func(c byte) {
		if c == '\n' {
			line, col = line+1, 1
		} else {
			col++
		}
	}
	for i := 0; i < len(src); {
		c := src[i]
		switch {
		case strings.HasPrefix(src[i:], "/*"):
			end := strings.Index(src[i+2:], "*/")
			n := len(src) - i
			if end >= 0 {
				n = end + 4
			}
			for _, b := range []byte(src[i : i+n]) {
				advance(b)
			}
			i += n
			continue
		case strings.HasPrefix(src[i:], "//") || c == '#':
			for i < len(src) && src[i] != '\n' {
				advance(src[i])
				i++
			}
			continue
		case c == '"' || c == '\'':
			advance(c)
			i++
			for i < len(src) && src[i] != c {
				if src[i] == '\\' && i+1 < len(src) && (src[i+1] == '"' || src[i+1] == '\'') {
					advance(src[i])
					i++
				}
				advance(src[i])
				i++
			}
			if i < len(src) {
				advance(src[i])
				i++
			}
			continue
		case c == '{' || c == '(':
			depth++
		case c == '}' || c == ')':
			depth--
		case isWordStart(c) && (i == 0 || !isWordByte(src[i-1])):
			j := i
			for j < len(src) && isWordByte(src[j]) {
				j++
			}
			word := src[i:j]
			if depth == 0 {
				kinds[word] = append(kinds[word], ast.Position{Filename: r.path, Line: line, Col: col})
			}
			for _, b := range []byte(word) {
				advance(b)
			}
			i = j
			continue
		}
		advance(c)
		i++
	}

	out := map[any]ast.Position{}
	place := func(kind string, n int, node any) {
		if n < len(kinds[kind]) {
			out[node] = kinds[kind][n]
		}
	}
	for n, td := range r.t.Typedefs {
		place("typedef", n, td)
	}
	for n, c := range r.t.Constants {
		place("const", n, c)
	}
	for n, e := range r.t.Enums {
		place("enum", n, e)
	}
	for n, s := range r.t.Structs {
		place("struct", n, s)
	}
	for n, u := range r.t.Unions {
		place("union", n, u)
	}
	for n, x := range r.t.Exceptions {
		place("exception", n, x)
	}
	for n, svc := range r.t.Services {
		place("service", n, svc)
	}
	return out
}

func isWordStart(c byte) bool {
	return c == '_' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

func isWordByte(c byte) bool {
	return isWordStart(c) || (c >= '0' && c <= '9') || c == '.'
}

// get is an annotation's value, decoded, and whether it is present.
func get(anns parser.Annotations, key string) (string, bool) {
	for _, a := range anns {
		if a.Key == key && len(a.Values) > 0 {
			return unliteral.Replace(a.Values[0]), true
		}
	}
	return "", false
}

// all is every value of a repeated annotation, decoded.
func all(anns parser.Annotations, key string) []string {
	var out []string
	for _, a := range anns {
		if a.Key == key {
			for _, v := range a.Values {
				out = append(out, unliteral.Replace(v))
			}
		}
	}
	return out
}

// Normalize puts .thrift files into the normal form the round-trip corpus
// compares: what thriftgo parses, without comments, with each kind of
// definition sorted by name, as JSON.
func Normalize(files []*plugin.File) ([]*plugin.File, error) {
	out := make([]*plugin.File, len(files))
	for i, f := range files {
		t, err := parser.ParseString(f.GetPath(), string(f.GetContent()))
		if err != nil {
			return nil, fmt.Errorf("%s: %w", f.GetPath(), err)
		}
		t.Filename = ""
		slices.SortFunc(t.Typedefs, func(a, b *parser.Typedef) int { return cmp.Compare(a.Alias, b.Alias) })
		slices.SortFunc(t.Constants, func(a, b *parser.Constant) int { return cmp.Compare(a.Name, b.Name) })
		slices.SortFunc(t.Enums, func(a, b *parser.Enum) int { return cmp.Compare(a.Name, b.Name) })
		for _, list := range [][]*parser.StructLike{t.Structs, t.Unions, t.Exceptions} {
			slices.SortFunc(list, func(a, b *parser.StructLike) int { return cmp.Compare(a.Name, b.Name) })
		}
		slices.SortFunc(t.Services, func(a, b *parser.Service) int { return cmp.Compare(a.Name, b.Name) })
		data, err := json.Marshal(t)
		if err != nil {
			return nil, err
		}
		var tree any
		if err := json.Unmarshal(data, &tree); err != nil {
			return nil, err
		}
		data, err = json.MarshalIndent(withoutComments(tree), "", "  ")
		if err != nil {
			return nil, err
		}
		out[i] = &plugin.File{Path: f.GetPath(), Content: append(data, '\n')}
	}
	return out, nil
}

// withoutComments drops every ReservedComments key from a JSON tree.
func withoutComments(v any) any {
	switch v := v.(type) {
	case map[string]any:
		delete(v, "ReservedComments")
		for k, e := range v {
			v[k] = withoutComments(e)
		}
	case []any:
		for i, e := range v {
			v[i] = withoutComments(e)
		}
	}
	return v
}
