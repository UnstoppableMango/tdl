package protobuf

import (
	"cmp"
	"context"
	"errors"
	"path"
	"slices"
	"strconv"
	"strings"

	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"

	"github.com/unstoppablemango/tdl/ast"
	"github.com/unstoppablemango/tdl/backend/internal/emit"
	"github.com/unstoppablemango/tdl/backend/internal/reverse"
	"github.com/unstoppablemango/tdl/ir"
	"github.com/unstoppablemango/tdl/plugin"
)

// Import reads .proto files of one package into a model: each message a
// struct, each enum an enum, and a message holding only a oneof of its own
// nested messages, as Generate writes a fielded enum, an enum with fields.
// What regenerating needs goes in a protobuf target block. Under the
// annotations a `roundtrip` directive writes, it rebuilds the model the
// files were generated from. docs/design/schema-backends.md has the
// mapping.
func (Backend) Import(ctx context.Context, req *plugin.ImportRequest) (*plugin.ImportResponse, error) {
	r := &reader{
		names:   map[string]string{},
		taken:   map[string]bool{},
		foreign: map[string]string{},
		uses:    map[string]map[string]bool{},
	}
	return r.Response(r.read(ctx, req.GetFiles()))
}

type reader struct {
	reverse.Reader
	c   *compiled
	pkg string

	// files is the files being imported, by path.
	files []*protoFile

	// names maps each imported message's and enum's full name, with its
	// leading dot, to its TDL name.
	names map[string]string
	taken map[string]bool
	// foreign maps a message from an imported file to the declaration
	// standing in for it.
	foreign map[string]string
	// uses records, per file, the files its types are declared in.
	uses map[string]map[string]bool

	// messages and enums index every compiled file's types by full name.
	messages map[string]*descriptorpb.DescriptorProto
	enums    map[string]*descriptorpb.EnumDescriptorProto
	declared map[string]string // full name to the file declaring it

	// extra holds what reading the current declaration added: hoisted
	// nested types and stand-ins for foreign ones.
	extra []ast.Decl
}

// protoFile is one file being imported, with its comments by path.
type protoFile struct {
	fd   *descriptorpb.FileDescriptorProto
	locs map[string]*descriptorpb.SourceCodeInfo_Location
}

func (s *protoFile) loc(p ...int32) *descriptorpb.SourceCodeInfo_Location {
	return s.locs[pathKey(p)]
}

func (s *protoFile) pos(p ...int32) *ir.Position {
	l := s.loc(p...)
	if l == nil || len(l.GetSpan()) < 2 {
		return &ir.Position{Filename: s.fd.GetName()}
	}
	return &ir.Position{Filename: s.fd.GetName(), Line: l.GetSpan()[0] + 1, Column: l.GetSpan()[1] + 1}
}

func (s *protoFile) astPos(p ...int32) ast.Position {
	ip := s.pos(p...)
	return ast.Position{Filename: ip.GetFilename(), Line: int(ip.GetLine()), Col: int(ip.GetColumn())}
}

func pathKey(p []int32) string {
	var b strings.Builder
	for _, n := range p {
		b.WriteString(strconv.Itoa(int(n)))
		b.WriteByte('.')
	}
	return b.String()
}

func child(p []int32, more ...int32) []int32 {
	return append(slices.Clone(p), more...)
}

// Field numbers in descriptor.proto, which source info paths are made of.
const (
	filePackage = 2
	fileMessage = 4
	fileEnum    = 5
	fileService = 6
	msgField    = 2
	msgNested   = 3
	msgEnum     = 4
	msgOneof    = 8
	enumValue   = 2
)

func (r *reader) read(ctx context.Context, files []*plugin.File) (*ir.Model, error) {
	sources := map[string]string{}
	var paths []string
	for _, f := range files {
		if f.GetPath() == annotationsFile {
			continue
		}
		sources[f.GetPath()] = string(f.GetContent())
		paths = append(paths, f.GetPath())
	}
	if len(paths) == 0 {
		return nil, reverse.Failf(nil, "no .proto files to import")
	}
	slices.Sort(paths)

	c, err := compile(ctx, sources, paths, true)
	var bad *compileError
	if errors.As(err, &bad) {
		return nil, reverse.Failf(nil, "the files do not compile:\n%s", bad.text)
	}
	if err != nil {
		return nil, err
	}
	r.c = c
	r.index()

	for _, p := range paths {
		fd := c.file(p)
		s := &protoFile{fd: fd, locs: map[string]*descriptorpb.SourceCodeInfo_Location{}}
		for _, l := range fd.GetSourceCodeInfo().GetLocation() {
			if k := pathKey(l.GetPath()); s.locs[k] == nil {
				s.locs[k] = l
			}
		}
		r.files = append(r.files, s)
		if r.ext(fd.GetOptions(), "file") != nil {
			r.Roundtrip = true
		}
	}

	r.pkg = r.files[0].fd.GetPackage()
	for _, s := range r.files[1:] {
		if s.fd.GetPackage() != r.pkg {
			return nil, reverse.Failf(s.pos(filePackage), "%s declares package %s and %s declares %s; import one package at a time",
				r.files[0].fd.GetName(), r.pkg, s.fd.GetName(), s.fd.GetPackage())
		}
	}

	file := &ast.File{Filename: paths[0]}
	tdlPkg := r.pkg
	var items []reverse.Item
	for _, s := range r.files {
		ann := r.ext(s.fd.GetOptions(), "file")
		if ann == nil {
			continue
		}
		if p := str(ann, "package"); p != "" {
			tdlPkg = p
		}
		for _, src := range strs(ann, "imports") {
			imp, err := reverse.ParseImport(src)
			if err != nil {
				return nil, reverse.Failf(s.pos(), "(tdl.file) import %q is not one TDL import: %v", src, err)
			}
			file.Imports = append(file.Imports, imp)
		}
		for _, it := range list(ann, "items") {
			decl, err := reverse.ParseItem(str(it, "source"))
			if err != nil {
				return nil, reverse.Failf(s.pos(), "a (tdl.file) item is not TDL: %v", err)
			}
			items = append(items, reverse.Item{Decl: decl, At: int(num(it, "at"))})
		}
	}
	var doc []string
	for _, s := range r.files {
		if doc = comments(s.loc(filePackage)); len(doc) > 0 {
			break
		}
	}
	if tdlPkg != "" || len(doc) > 0 {
		file.Package = &ast.PackageDecl{Path: tdlPkg, Doc: doc, DocP: make([]ast.Position, len(doc))}
	}

	r.nameAll()

	var emitted []reverse.Item
	for _, s := range r.files {
		decls, err := r.file(s)
		if err != nil {
			return nil, err
		}
		emitted = append(emitted, decls...)
	}
	if !r.Roundtrip {
		r.block(file)
	}

	file.Decls = reverse.Arrange(emitted, items)
	if len(r.Entries) > 0 {
		file.Decls = append(file.Decls, &ast.TargetDecl{DeclHead: ast.DeclHead{N: Name}, For: tdlPkg, Entries: r.Entries})
	}

	return reverse.Lower(file)
}

// index records every compiled message and enum by full name.
func (r *reader) index() {
	r.messages = map[string]*descriptorpb.DescriptorProto{}
	r.enums = map[string]*descriptorpb.EnumDescriptorProto{}
	r.declared = map[string]string{}
	var walk func(file, scope string, msgs []*descriptorpb.DescriptorProto, enums []*descriptorpb.EnumDescriptorProto)
	walk = func(file, scope string, msgs []*descriptorpb.DescriptorProto, enums []*descriptorpb.EnumDescriptorProto) {
		for _, m := range msgs {
			full := scope + "." + m.GetName()
			r.messages[full], r.declared[full] = m, file
			walk(file, full, m.GetNestedType(), m.GetEnumType())
		}
		for _, e := range enums {
			full := scope + "." + e.GetName()
			r.enums[full], r.declared[full] = e, file
		}
	}
	for _, f := range r.c.files {
		scope := ""
		if f.GetPackage() != "" {
			scope = "." + f.GetPackage()
		}
		walk(f.GetName(), scope, f.GetMessageType(), f.GetEnumType())
	}
}

func (r *reader) scope() string {
	if r.pkg == "" {
		return ""
	}
	return "." + r.pkg
}

// nameAll gives every message and enum being imported its TDL name before
// any is read, since a field may name one declared after it. A nested type
// is hoisted to the top level, under its own name when that is free.
func (r *reader) nameAll() {
	for _, s := range r.files {
		for _, m := range s.fd.GetMessageType() {
			r.claim(r.scope()+"."+m.GetName(), r.declName(m.GetOptions(), "message", m.GetName()))
		}
		for _, e := range s.fd.GetEnumType() {
			r.claim(r.scope()+"."+e.GetName(), r.declName(e.GetOptions(), "enum", e.GetName()))
		}
	}
	var nested func(full, owner string, m *descriptorpb.DescriptorProto)
	nested = func(full, owner string, m *descriptorpb.DescriptorProto) {
		sum := r.isSum(full, m)
		for _, n := range m.GetNestedType() {
			if sum || n.GetOptions().GetMapEntry() {
				continue
			}
			inner := full + "." + n.GetName()
			r.claim(inner, r.free(n.GetName(), owner))
			nested(inner, r.names[inner], n)
		}
		for _, e := range m.GetEnumType() {
			r.claim(full+"."+e.GetName(), r.free(e.GetName(), owner))
		}
	}
	for _, s := range r.files {
		for _, m := range s.fd.GetMessageType() {
			full := r.scope() + "." + m.GetName()
			nested(full, r.names[full], m)
		}
	}
}

func (r *reader) claim(full, name string) {
	r.names[full] = name
	r.taken[name] = true
}

// free is name when no declaration has it, and otherwise name prefixed
// with owner's.
func (r *reader) free(name, owner string) string {
	if !r.taken[name] {
		return name
	}
	for n := owner + name; ; n += "_" {
		if !r.taken[n] {
			return n
		}
	}
}

// declName is a message's or enum's TDL name: its annotation's, or its own.
func (r *reader) declName(opts proto.Message, ext, name string) string {
	if n := str(r.ext(opts, ext), "name"); n != "" {
		return n
	}
	return name
}

// isSum reports whether a message is how Generate writes a fielded enum:
// only a oneof named variant, holding one member per nested message, each
// named for its message.
func (r *reader) isSum(full string, m *descriptorpb.DescriptorProto) bool {
	if len(m.GetField()) == 0 || len(m.GetOneofDecl()) != 1 || m.GetOneofDecl()[0].GetName() != oneof ||
		len(m.GetNestedType()) != len(m.GetField()) || len(m.GetEnumType()) > 0 {
		return false
	}
	for _, f := range m.GetField() {
		if f.OneofIndex == nil || f.GetProto3Optional() || f.GetType() != descriptorpb.FieldDescriptorProto_TYPE_MESSAGE {
			return false
		}
		inner, ok := strings.CutPrefix(f.GetTypeName(), full+".")
		if !ok || strings.Contains(inner, ".") || emit.Snake(inner) != f.GetName() {
			return false
		}
	}
	return true
}

// file reads one file's top-level declarations in source order.
func (r *reader) file(s *protoFile) ([]reverse.Item, error) {
	type top struct {
		line int32
		read func() (*reverse.Item, error)
	}
	var tops []top
	line := func(p ...int32) int32 { return s.pos(p...).GetLine() }
	for i, m := range s.fd.GetMessageType() {
		p := []int32{fileMessage, int32(i)}
		tops = append(tops, top{line(p...), func() (*reverse.Item, error) { return r.message(s, p, r.scope(), m) }})
	}
	for i, e := range s.fd.GetEnumType() {
		p := []int32{fileEnum, int32(i)}
		tops = append(tops, top{line(p...), func() (*reverse.Item, error) {
			return r.enum(s, p, r.scope()+"."+e.GetName(), e), nil
		}})
	}
	for i, svc := range s.fd.GetService() {
		p := []int32{fileService, int32(i)}
		tops = append(tops, top{line(p...), func() (*reverse.Item, error) { return r.service(s, p, svc) }})
	}
	slices.SortStableFunc(tops, func(a, b top) int { return cmp.Compare(a.line, b.line) })

	var out []reverse.Item
	for _, t := range tops {
		r.extra = nil
		it, err := t.read()
		if err != nil {
			return nil, err
		}
		if it == nil {
			continue
		}
		it.Extra = r.extra
		out = append(out, *it)
		if !r.Roundtrip && len(r.files) > 1 {
			r.Directive(it.Decl.Name(), "file", reverse.StrLit(path.Base(s.fd.GetName())))
		}
	}
	return out, nil
}

// at is the index an annotation gives a declaration, or -1.
func at(ann protoreflect.Message) int {
	if ann == nil || !has(ann, "at") {
		return -1
	}
	return int(num(ann, "at"))
}

// comments is a location's leading comment, one line per entry, each with
// the space after `//` removed.
func comments(l *descriptorpb.SourceCodeInfo_Location) []string {
	c := l.GetLeadingComments()
	if c == "" {
		return nil
	}
	lines := strings.Split(strings.TrimSuffix(c, "\n"), "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(strings.TrimPrefix(line, " "), " \t")
	}
	return lines
}

// message reads a top-level or hoisted message: a struct, or an enum with
// fields when it is written as one.
func (r *reader) message(s *protoFile, p []int32, scope string, m *descriptorpb.DescriptorProto) (*reverse.Item, error) {
	full := scope + "." + m.GetName()
	name := r.names[full]
	ann := r.ext(m.GetOptions(), "message")
	if !r.Roundtrip && emit.Pascal(m.GetName()) != m.GetName() {
		r.Directive(name, "name", reverse.StrLit(m.GetName()))
	}
	if r.isSum(full, m) {
		decl, err := r.sum(s, p, full, name, m)
		if err != nil {
			return nil, err
		}
		return &reverse.Item{Decl: decl, At: at(ann)}, nil
	}

	decl := &ast.StructDecl{
		DeclHead: reverse.Head(name, comments(s.loc(p...)), s.astPos(p...), m.GetOptions().GetDeprecated()),
		Keyword:  "type",
	}
	switch enumValueName(ann, "kind") {
	case "KIND_ENTITY":
		decl.Conforms = []*ast.ClassRef{{N: "Entity"}}
	case "KIND_MIXIN":
		decl.Keyword = "mixin"
	}
	if src := str(ann, "conforms"); src != "" {
		conforms, err := reverse.ParseConforms(src)
		if err != nil {
			return nil, reverse.Failf(s.pos(p...), "the (tdl.message) conforms of %s is not a TDL conformance list: %v", m.GetName(), err)
		}
		decl.Conforms = conforms
	}
	skip := r.reserved(s, p, name, m)
	r.options(m.GetOptions(), name)

	var slots []slot
	include := ""
	seen := map[int32]bool{}
	for i, f := range m.GetField() {
		fp := child(p, msgField, int32(i))
		if f.OneofIndex != nil && !f.GetProto3Optional() {
			k := f.GetOneofIndex()
			if seen[k] {
				continue
			}
			seen[k] = true
			field, ns, err := r.oneof(s, p, k, name, m)
			if err != nil {
				return nil, err
			}
			slots = append(slots, ns...)
			decl.Members = r.member(decl.Members, field, r.ext(m.GetOneofDecl()[k].GetOptions(), "oneof"), &include)
			continue
		}
		field, err := r.field(s, fp, f, name)
		if err != nil {
			return nil, err
		}
		if field == nil {
			continue
		}
		slots = append(slots, slot{Member: emit.Member{Name: name + "." + field.N}, num: int64(f.GetNumber())})
		decl.Members = r.member(decl.Members, field, r.ext(f.GetOptions(), "field"), &include)
	}
	r.pins(slots, fieldNumbers, skip)

	for i, n := range m.GetNestedType() {
		if n.GetOptions().GetMapEntry() {
			continue
		}
		np := child(p, msgNested, int32(i))
		r.Warn(emit.LossUnsupported, s.pos(np...), "%s is nested in %s, and is read as %s at the top level", n.GetName(), m.GetName(), r.names[full+"."+n.GetName()])
		it, err := r.message(s, np, full, n)
		if err != nil {
			return nil, err
		}
		r.extra = append(r.extra, it.Decl)
	}
	for i, e := range m.GetEnumType() {
		ep := child(p, msgEnum, int32(i))
		r.Warn(emit.LossUnsupported, s.pos(ep...), "%s is nested in %s, and is read as %s at the top level", e.GetName(), m.GetName(), r.names[full+"."+e.GetName()])
		r.extra = append(r.extra, r.enum(s, ep, full+"."+e.GetName(), e).Decl)
	}
	if len(m.GetExtension()) > 0 || len(m.GetExtensionRange()) > 0 {
		r.Warn(emit.LossUnsupported, s.pos(p...), "%s declares extensions, which TDL has no form for", m.GetName())
	}
	return &reverse.Item{Decl: decl, At: at(ann)}, nil
}

// member adds a field to a struct body, or the include that copied it.
// Consecutive fields one include copied become one include.
func (r *reader) member(members []ast.Member, f *ast.Field, ann protoreflect.Message, include *string) []ast.Member {
	inc := str(ann, "include")
	if inc == "" {
		*include = ""
		return append(members, f)
	}
	if inc != *include {
		*include = inc
		members = append(members, &ast.Include{P: f.P, Type: &ast.ClassRef{N: inc}})
	}
	return members
}

// pins writes a number directive on each slot whose number allocation
// would not give it.
func (r *reader) pins(slots []slot, rule emit.NumberRule, skip map[int64]bool) {
	if r.Roundtrip || len(slots) == 0 {
		return
	}
	rule.Skip = skip
	nums := make([]int64, len(slots))
	for i, s := range slots {
		nums[i] = s.num
	}
	for i, pinned := range emit.Pins(nums, rule) {
		if pinned {
			r.Directive(slots[i].Name, "number", &ast.Literal{Kind: ast.LitInt, Text: strconv.FormatInt(slots[i].num, 10)})
		}
	}
}

// reserved writes a message's reserved numbers and its reserved names as
// a directive each, and returns the numbers.
func (r *reader) reserved(s *protoFile, p []int32, name string, m *descriptorpb.DescriptorProto) map[int64]bool {
	numbers := map[int64]bool{}
	var args []*ast.Literal
	for _, rr := range m.GetReservedRange() {
		if rr.GetEnd()-rr.GetStart() > 64 {
			r.Warn(emit.LossUnsupported, s.pos(p...), "%s reserves %d to %d, and a reserved directive lists numbers one by one", m.GetName(), rr.GetStart(), rr.GetEnd()-1)
			continue
		}
		for n := rr.GetStart(); n < rr.GetEnd(); n++ {
			numbers[int64(n)] = true
			args = append(args, &ast.Literal{Kind: ast.LitInt, Text: strconv.Itoa(int(n))})
		}
	}
	// protobuf keeps reserved numbers and names in separate statements.
	if len(args) > 0 {
		r.Directive(name, "reserved", args...)
	}
	var names []*ast.Literal
	for _, n := range m.GetReservedName() {
		names = append(names, reverse.StrLit(n))
	}
	if len(names) > 0 {
		r.Directive(name, "reserved", names...)
	}
	return numbers
}

// field reads one field: from its annotation under roundtrip, and
// otherwise from its name and type. It returns nil for a field it skips.
func (r *reader) field(s *protoFile, p []int32, f *descriptorpb.FieldDescriptorProto, owner string) (*ast.Field, error) {
	pos := s.astPos(p...)
	var out *ast.Field
	if src := str(r.ext(f.GetOptions(), "field"), "source"); src != "" {
		parsed, err := reverse.ParseField(src)
		if err != nil {
			return nil, reverse.Failf(s.pos(p...), "the (tdl.field) source of %s.%s is not a TDL field: %v", owner, f.GetName(), err)
		}
		out = parsed
	} else {
		typ := r.typeRef(s, p, f)
		if typ == nil {
			return nil, nil
		}
		out = &ast.Field{Type: typ}
		out.N = emit.Camel(f.GetName())
		if !r.Roundtrip && emit.Snake(out.N) != f.GetName() {
			r.Directive(owner+"."+out.N, "name", reverse.StrLit(f.GetName()))
		}
	}
	h := reverse.Head(out.N, comments(s.loc(p...)), pos, f.GetOptions().GetDeprecated())
	out.DeclHead = h
	r.fieldOptions(f, owner+"."+out.N)
	if t := s.loc(p...).GetTrailingComments(); t != "" && !r.Roundtrip {
		r.Warn(emit.LossDoc, s.pos(p...), "%s.%s's trailing comment is not a doc comment, and is dropped", owner, out.N)
	}
	return out, nil
}

// fieldOptions writes a field's options other than deprecated, and a JSON
// name protobuf would not derive, as option directives.
func (r *reader) fieldOptions(f *descriptorpb.FieldDescriptorProto, at string) {
	if r.Roundtrip {
		return
	}
	r.options(f.GetOptions(), at)
	if j := f.GetJsonName(); j != "" && j != jsonName(f.GetName()) {
		r.Directive(at, "option", reverse.StrLit("json_name"), reverse.StrLit(quote(j)))
	}
}

// jsonName is the JSON name protobuf derives from a field's name.
func jsonName(name string) string {
	var b strings.Builder
	upper := false
	for _, c := range name {
		if c == '_' {
			upper = true
			continue
		}
		if upper && c >= 'a' && c <= 'z' {
			c -= 'a' - 'A'
		}
		upper = false
		b.WriteRune(c)
	}
	return b.String()
}

// oneof reads a oneof outside a fielded enum: the field Generate inlines
// with a `oneof` directive. Without annotations its enum is made up: one
// variant per member, each carrying that member as its one field.
func (r *reader) oneof(s *protoFile, p []int32, k int32, owner string, m *descriptorpb.DescriptorProto) (*ast.Field, []slot, error) {
	o := m.GetOneofDecl()[k]
	op := child(p, msgOneof, k)
	var members []int
	for i, f := range m.GetField() {
		if f.OneofIndex != nil && f.GetOneofIndex() == k {
			members = append(members, i)
		}
	}

	if src := str(r.ext(o.GetOptions(), "oneof"), "source"); src != "" {
		field, err := reverse.ParseField(src)
		if err != nil {
			return nil, nil, reverse.Failf(s.pos(op...), "the (tdl.oneof) source of %s.%s is not a TDL field: %v", owner, o.GetName(), err)
		}
		dep := field.Dep
		field.DeclHead = reverse.Head(field.N, comments(s.loc(op...)), s.astPos(op...), false)
		field.Dep = dep
		return field, nil, nil
	}

	enumName := r.free(emit.Pascal(o.GetName()), owner)
	r.taken[enumName] = true
	fieldName := emit.Camel(o.GetName())
	if emit.Snake(fieldName) != o.GetName() {
		r.Warn(emit.LossName, s.pos(op...), "oneof %s regenerates as %s", o.GetName(), emit.Snake(fieldName))
	}
	e := &ast.EnumDecl{DeclHead: ast.DeclHead{N: enumName, P: s.astPos(op...)}}
	var slots []slot
	for _, i := range members {
		f := m.GetField()[i]
		fp := child(p, msgField, int32(i))
		variant := emit.Pascal(f.GetName())
		vf, err := r.field(s, fp, f, enumName+"."+variant)
		if err != nil {
			return nil, nil, err
		}
		if vf == nil {
			continue
		}
		if emit.Snake(vf.N) != f.GetName() {
			r.Warn(emit.LossName, s.pos(fp...), "oneof member %s regenerates as %s", f.GetName(), emit.Snake(vf.N))
		}
		e.Variants = append(e.Variants, &ast.Variant{DeclHead: ast.DeclHead{N: variant, P: vf.P}, Fields: []*ast.Field{vf}, Rbrace: vf.P})
		slots = append(slots, slot{Member: emit.Member{Name: enumName + "." + variant}, num: int64(f.GetNumber())})
	}
	r.extra = append(r.extra, e)
	if hasOptions(o.GetOptions()) {
		r.Warn(emit.LossUnsupported, s.pos(op...), "oneof %s's options have no TDL form", o.GetName())
	}

	field := &ast.Field{DeclHead: reverse.Head(fieldName, comments(s.loc(op...)), s.astPos(op...), false), Type: &ast.TypeRef{N: enumName}}
	r.Entries = append(r.Entries, &ast.TargetEntry{Path: owner + "." + fieldName, Directive: &ast.Directive{N: "oneof"}})
	return field, slots, nil
}

// sum reads a message written as a fielded enum.
func (r *reader) sum(s *protoFile, p []int32, full, name string, m *descriptorpb.DescriptorProto) (*ast.EnumDecl, error) {
	decl := &ast.EnumDecl{DeclHead: reverse.Head(name, comments(s.loc(p...)), s.astPos(p...), m.GetOptions().GetDeprecated())}
	r.options(m.GetOptions(), name)
	var slots []slot
	for _, f := range m.GetField() {
		inner := strings.TrimPrefix(f.GetTypeName(), full+".")
		j := slices.IndexFunc(m.GetNestedType(), func(n *descriptorpb.DescriptorProto) bool { return n.GetName() == inner })
		n := m.GetNestedType()[j]
		np := child(p, msgNested, int32(j))
		vname := r.declName(n.GetOptions(), "message", n.GetName())
		path := name + "." + vname
		if !r.Roundtrip && emit.Pascal(n.GetName()) != n.GetName() {
			r.Directive(path, "name", reverse.StrLit(n.GetName()))
		}
		v := &ast.Variant{DeclHead: reverse.Head(vname, comments(s.loc(np...)), s.astPos(np...), f.GetOptions().GetDeprecated())}
		if !r.Roundtrip && hasOptions(f.GetOptions(), "deprecated") {
			r.Warn(emit.LossUnsupported, s.pos(np...), "%s's member %s carries options a variant has no form for", m.GetName(), f.GetName())
		}
		var fields []slot
		for k, vf := range n.GetField() {
			field, err := r.field(s, child(np, msgField, int32(k)), vf, path)
			if err != nil {
				return nil, err
			}
			if field != nil {
				v.Fields = append(v.Fields, field)
				fields = append(fields, slot{Member: emit.Member{Name: path + "." + field.N}, num: int64(vf.GetNumber())})
			}
		}
		if len(v.Fields) > 0 {
			v.Rbrace = v.P
		}
		r.pins(fields, fieldNumbers, nil)
		if len(n.GetOneofDecl()) > 0 || len(n.GetNestedType()) > 0 || len(n.GetEnumType()) > 0 {
			r.Warn(emit.LossUnsupported, s.pos(np...), "variant %s holds a oneof or a nested type, which are not read", vname)
		}
		decl.Variants = append(decl.Variants, v)
		slots = append(slots, slot{Member: emit.Member{Name: path}, num: int64(f.GetNumber())})
	}
	r.pins(slots, fieldNumbers, nil)
	return decl, nil
}

// enum reads an enum, dropping the zero value Generate writes first.
func (r *reader) enum(s *protoFile, p []int32, full string, e *descriptorpb.EnumDescriptorProto) *reverse.Item {
	name := r.names[full]
	ann := r.ext(e.GetOptions(), "enum")
	decl := &ast.EnumDecl{DeclHead: reverse.Head(name, comments(s.loc(p...)), s.astPos(p...), e.GetOptions().GetDeprecated())}
	if !r.Roundtrip {
		if emit.Pascal(e.GetName()) != e.GetName() {
			r.Directive(name, "name", reverse.StrLit(e.GetName()))
		}
		r.options(e.GetOptions(), name, "allow_alias")
		if e.GetOptions().GetAllowAlias() {
			r.Warn(emit.LossUnsupported, s.pos(p...), "%s allows aliases, which TDL has no form for", e.GetName())
		}
		if len(e.GetReservedRange()) > 0 || len(e.GetReservedName()) > 0 {
			r.Warn(emit.LossUnsupported, s.pos(p...), "%s reserves values, which TDL has no form for", e.GetName())
		}
	}
	prefix := emit.ScreamingSnake(e.GetName())
	var slots []slot
	for i, v := range e.GetValue() {
		vp := child(p, enumValue, int32(i))
		if v.GetNumber() == 0 {
			if v.GetName() != prefix+"_UNSPECIFIED" && !r.Roundtrip {
				r.Warn(emit.LossName, s.pos(vp...), "%s's zero value %s regenerates as %s_UNSPECIFIED", e.GetName(), v.GetName(), prefix)
			}
			continue
		}
		vname, renamed := valueBack(prefix, v.GetName())
		if n := str(r.ext(v.GetOptions(), "value"), "name"); n != "" {
			vname = n
		}
		path := name + "." + vname
		if !r.Roundtrip {
			if renamed {
				r.Directive(path, "name", reverse.StrLit(v.GetName()))
			}
			r.options(v.GetOptions(), path)
		}
		decl.Variants = append(decl.Variants, &ast.Variant{DeclHead: reverse.Head(vname, comments(s.loc(vp...)), s.astPos(vp...), v.GetOptions().GetDeprecated())})
		slots = append(slots, slot{Member: emit.Member{Name: path}, num: int64(v.GetNumber())})
	}
	r.pins(slots, enumNumbers, nil)
	return &reverse.Item{Decl: decl, At: at(ann)}
}

// service reads a service, which only an annotation carries yet.
func (r *reader) service(s *protoFile, p []int32, svc *descriptorpb.ServiceDescriptorProto) (*reverse.Item, error) {
	ann := r.ext(svc.GetOptions(), "service")
	src := str(ann, "source")
	if src == "" {
		r.Warn(emit.LossUnsupported, s.pos(p...), "service %s is not imported yet", svc.GetName())
		return nil, nil
	}
	decl, err := reverse.ParseItem(src)
	if err != nil {
		return nil, reverse.Failf(s.pos(p...), "the (tdl.service) source of %s is not TDL: %v", svc.GetName(), err)
	}
	return &reverse.Item{Decl: decl, At: at(ann)}, nil
}

var wrappers = map[string]string{
	".google.protobuf.DoubleValue": "float64",
	".google.protobuf.FloatValue":  "float32",
	".google.protobuf.Int64Value":  "int64",
	".google.protobuf.UInt64Value": "uint64",
	".google.protobuf.Int32Value":  "int32",
	".google.protobuf.UInt32Value": "uint32",
	".google.protobuf.BoolValue":   "bool",
	".google.protobuf.StringValue": "string",
	".google.protobuf.BytesValue":  "bytes",
}

// typeRef reads a field's type, or warns and returns nil.
func (r *reader) typeRef(s *protoFile, p []int32, f *descriptorpb.FieldDescriptorProto) *ast.TypeRef {
	if f.GetType() == descriptorpb.FieldDescriptorProto_TYPE_MESSAGE {
		if entry := r.messages[f.GetTypeName()]; entry.GetOptions().GetMapEntry() && len(entry.GetField()) == 2 {
			k, v := r.single(s, p, entry.GetField()[0]), r.single(s, p, entry.GetField()[1])
			if k == nil || v == nil {
				return nil
			}
			return &ast.TypeRef{MapKey: k, MapValue: v}
		}
	}
	t := r.single(s, p, f)
	if t == nil {
		return nil
	}
	if f.GetLabel() == descriptorpb.FieldDescriptorProto_LABEL_REPEATED {
		return &ast.TypeRef{List: t}
	}
	if f.GetLabel() == descriptorpb.FieldDescriptorProto_LABEL_REQUIRED {
		r.Warn(emit.LossUnsupported, s.pos(p...), "%s is required, which regenerates as an ordinary field", f.GetName())
	}
	if f.GetProto3Optional() {
		t.Optional = true
	}
	return t
}

var scalarsBack = map[descriptorpb.FieldDescriptorProto_Type]string{
	descriptorpb.FieldDescriptorProto_TYPE_DOUBLE: "float64",
	descriptorpb.FieldDescriptorProto_TYPE_FLOAT:  "float32",
	descriptorpb.FieldDescriptorProto_TYPE_INT64:  "int64",
	descriptorpb.FieldDescriptorProto_TYPE_UINT64: "uint64",
	descriptorpb.FieldDescriptorProto_TYPE_INT32:  "int32",
	descriptorpb.FieldDescriptorProto_TYPE_UINT32: "uint32",
	descriptorpb.FieldDescriptorProto_TYPE_BOOL:   "bool",
	descriptorpb.FieldDescriptorProto_TYPE_STRING: "string",
	descriptorpb.FieldDescriptorProto_TYPE_BYTES:  "bytes",
}

// encodings is the scalars that only change how an integer is encoded.
var encodings = map[descriptorpb.FieldDescriptorProto_Type]string{
	descriptorpb.FieldDescriptorProto_TYPE_SINT32:   "int32",
	descriptorpb.FieldDescriptorProto_TYPE_SINT64:   "int64",
	descriptorpb.FieldDescriptorProto_TYPE_SFIXED32: "int32",
	descriptorpb.FieldDescriptorProto_TYPE_SFIXED64: "int64",
	descriptorpb.FieldDescriptorProto_TYPE_FIXED32:  "uint32",
	descriptorpb.FieldDescriptorProto_TYPE_FIXED64:  "uint64",
}

// single reads a field's type without its label.
func (r *reader) single(s *protoFile, p []int32, f *descriptorpb.FieldDescriptorProto) *ast.TypeRef {
	if n, ok := scalarsBack[f.GetType()]; ok {
		return &ast.TypeRef{N: n}
	}
	if n, ok := encodings[f.GetType()]; ok {
		r.Warn(emit.LossPrimitive, s.pos(p...), "%s is %s, which regenerates as %s", f.GetName(), strings.ToLower(strings.TrimPrefix(f.GetType().String(), "TYPE_")), scalars[n])
		return &ast.TypeRef{N: n}
	}
	switch f.GetType() {
	case descriptorpb.FieldDescriptorProto_TYPE_MESSAGE, descriptorpb.FieldDescriptorProto_TYPE_ENUM:
	default:
		r.Warn(emit.LossUnsupported, s.pos(p...), "%s is a %s, which TDL has no form for", f.GetName(), f.GetType())
		return nil
	}

	full := f.GetTypeName()
	if file := r.declared[full]; file != "" {
		if r.uses[s.fd.GetName()] == nil {
			r.uses[s.fd.GetName()] = map[string]bool{}
		}
		r.uses[s.fd.GetName()][file] = true
	}
	if n, ok := r.names[full]; ok {
		return &ast.TypeRef{N: n}
	}
	switch full {
	case ".google.protobuf.Timestamp":
		return &ast.TypeRef{N: "instant"}
	case ".google.protobuf.Duration":
		return &ast.TypeRef{N: "duration"}
	}
	if n, ok := wrappers[full]; ok {
		r.Warn(emit.LossOptional, s.pos(p...), "%s is a %s, which regenerates as optional %s", f.GetName(), strings.TrimPrefix(full, "."), scalars[n])
		return &ast.TypeRef{N: n, Optional: true}
	}
	if r.Roundtrip {
		return &ast.TypeRef{N: emit.LastSegment(full)}
	}
	return &ast.TypeRef{N: r.stand(full)}
}

// stand is the declaration standing in for a message another file
// declares: an empty struct with a foreign directive naming it.
func (r *reader) stand(full string) string {
	if n, ok := r.foreign[full]; ok {
		return n
	}
	name := emit.LastSegment(full)
	if r.taken[name] {
		name = emit.Pascal(full)
	}
	r.foreign[full] = name
	r.taken[name] = true
	r.extra = append(r.extra, &ast.StructDecl{DeclHead: ast.DeclHead{N: name}, Keyword: "type"})
	r.Directive(name, "foreign", reverse.StrLit(r.declared[full]), reverse.StrLit(strings.TrimPrefix(full, ".")))
	return name
}

// block writes what regenerating the files needs on the target block: the
// edition, the file name, the file options, and imports nothing uses.
func (r *reader) block(file *ast.File) {
	first := r.files[0].fd
	var block []*ast.TargetEntry
	bare := func(name string, args ...*ast.Literal) {
		block = append(block, &ast.TargetEntry{Directive: &ast.Directive{N: name, Args: args}})
	}
	switch first.GetSyntax() {
	case "proto3":
	case "editions":
		bare("edition", reverse.StrLit(strings.TrimPrefix(first.GetEdition().String(), "EDITION_")))
	default:
		r.Warn(emit.LossUnsupported, r.files[0].pos(), "%s is proto2, and regenerates as proto3", first.GetName())
	}

	pkg := ""
	if file.Package != nil {
		pkg = file.Package.Path
	}
	want := filePath(pkg, "")
	for _, s := range r.files {
		if dir := path.Dir(s.fd.GetName()); dir != path.Dir(want) {
			r.Warn(emit.LossName, s.pos(), "%s regenerates in %s, the directory its package spells", s.fd.GetName(), path.Dir(want))
		}
	}
	if len(r.files) == 1 && path.Base(first.GetName()) != path.Base(want) {
		bare("file", reverse.StrLit(path.Base(first.GetName())))
	}

	for _, o := range r.optionList(first.GetOptions()) {
		bare("option", reverse.StrLit(o[0]), reverse.StrLit(o[1]))
	}

	var unused []string
	for _, s := range r.files {
		for _, dep := range s.fd.GetDependency() {
			if dep != annotationsFile && !r.uses[s.fd.GetName()][dep] && !slices.Contains(unused, dep) {
				unused = append(unused, dep)
			}
		}
	}
	for _, dep := range unused {
		bare("import", reverse.StrLit(dep))
	}
	r.Entries = append(block, r.Entries...)
}

// options writes a node's options as option directives, leaving out
// deprecated, which is a deprecation, and the annotations.
func (r *reader) options(opts proto.Message, at string, skip ...string) {
	for _, o := range r.optionList(opts, skip...) {
		r.Directive(at, "option", reverse.StrLit(o[0]), reverse.StrLit(o[1]))
	}
}

// optionList is an options message's set fields as name and value text,
// in field number order, one entry per element of a repeated option.
func (r *reader) optionList(opts proto.Message, skip ...string) [][2]string {
	if opts == nil {
		return nil
	}
	m := opts.ProtoReflect()
	if !m.IsValid() {
		return nil
	}
	type set struct {
		fd protoreflect.FieldDescriptor
		v  protoreflect.Value
	}
	var fields []set
	m.Range(func(fd protoreflect.FieldDescriptor, v protoreflect.Value) bool {
		fields = append(fields, set{fd, v})
		return true
	})
	slices.SortFunc(fields, func(a, b set) int { return cmp.Compare(a.fd.Number(), b.fd.Number()) })

	var out [][2]string
	for _, f := range fields {
		name := string(f.fd.Name())
		if f.fd.IsExtension() {
			if f.fd.ParentFile().Package() == "tdl" {
				continue
			}
			name = "(" + string(f.fd.FullName()) + ")"
		}
		if name == "deprecated" || name == "map_entry" || slices.Contains(skip, name) {
			continue
		}
		if f.fd.IsList() {
			l := f.v.List()
			for i := range l.Len() {
				out = append(out, [2]string{name, valueText(f.fd, l.Get(i))})
			}
			continue
		}
		out = append(out, [2]string{name, valueText(f.fd, f.v)})
	}
	return out
}

// hasOptions reports whether an options message sets anything but the
// annotations and the fields named.
func hasOptions(opts proto.Message, skip ...string) bool {
	if opts == nil || !opts.ProtoReflect().IsValid() {
		return false
	}
	found := false
	opts.ProtoReflect().Range(func(fd protoreflect.FieldDescriptor, _ protoreflect.Value) bool {
		if (fd.IsExtension() && fd.ParentFile().Package() == "tdl") || slices.Contains(skip, string(fd.Name())) {
			return true
		}
		found = true
		return false
	})
	return found
}

// valueText writes an option's value as protobuf source.
func valueText(fd protoreflect.FieldDescriptor, v protoreflect.Value) string {
	switch fd.Kind() {
	case protoreflect.StringKind:
		return quote(v.String())
	case protoreflect.BytesKind:
		return quote(string(v.Bytes()))
	case protoreflect.EnumKind:
		if ev := fd.Enum().Values().ByNumber(v.Enum()); ev != nil {
			return string(ev.Name())
		}
		return strconv.Itoa(int(v.Enum()))
	case protoreflect.MessageKind, protoreflect.GroupKind:
		text, _ := prototext.MarshalOptions{}.Marshal(v.Message().Interface())
		return "{" + strings.Join(strings.Fields(string(text)), " ") + "}"
	case protoreflect.FloatKind, protoreflect.DoubleKind:
		return strconv.FormatFloat(v.Float(), 'g', -1, 64)
	}
	return v.String()
}

// ext is the annotation an options message carries, or nil.
func (r *reader) ext(opts proto.Message, name string) protoreflect.Message {
	if opts == nil {
		return nil
	}
	m := opts.ProtoReflect()
	if !m.IsValid() {
		return nil
	}
	xt, err := r.c.types.FindExtensionByName(protoreflect.FullName("tdl." + name))
	if err != nil || !m.Has(xt.TypeDescriptor()) {
		return nil
	}
	return m.Get(xt.TypeDescriptor()).Message()
}

func field(m protoreflect.Message, name string) protoreflect.FieldDescriptor {
	if m == nil {
		return nil
	}
	return m.Descriptor().Fields().ByName(protoreflect.Name(name))
}

func has(m protoreflect.Message, name string) bool {
	fd := field(m, name)
	return fd != nil && m.Has(fd)
}

func str(m protoreflect.Message, name string) string {
	if fd := field(m, name); fd != nil {
		return m.Get(fd).String()
	}
	return ""
}

func num(m protoreflect.Message, name string) int64 {
	if fd := field(m, name); fd != nil {
		return m.Get(fd).Int()
	}
	return 0
}

func strs(m protoreflect.Message, name string) []string {
	fd := field(m, name)
	if fd == nil {
		return nil
	}
	l := m.Get(fd).List()
	out := make([]string, l.Len())
	for i := range out {
		out[i] = l.Get(i).String()
	}
	return out
}

func list(m protoreflect.Message, name string) []protoreflect.Message {
	fd := field(m, name)
	if fd == nil {
		return nil
	}
	l := m.Get(fd).List()
	out := make([]protoreflect.Message, l.Len())
	for i := range out {
		out[i] = l.Get(i).Message()
	}
	return out
}

func enumValueName(m protoreflect.Message, name string) string {
	fd := field(m, name)
	if fd == nil || !m.Has(fd) {
		return ""
	}
	if ev := fd.Enum().Values().ByNumber(m.Get(fd).Enum()); ev != nil {
		return string(ev.Name())
	}
	return ""
}
