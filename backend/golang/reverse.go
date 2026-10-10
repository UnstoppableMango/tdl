package golang

import (
	"bytes"
	"cmp"
	"context"
	"fmt"
	goast "go/ast"
	"go/format"
	goparser "go/parser"
	"go/token"
	"maps"
	"path"
	"slices"
	"strconv"
	"strings"
	"unicode"

	"github.com/unstoppablemango/tdl/ast"
	"github.com/unstoppablemango/tdl/backend/internal/emit"
	"github.com/unstoppablemango/tdl/backend/internal/reverse"
	"github.com/unstoppablemango/tdl/ir"
	"github.com/unstoppablemango/tdl/plugin"
)

// Import reads one Go package into a model: each struct a struct, a string
// type with constants of it an enum, an interface sealed by a marker method
// and implemented by structs named after it an enum with fields, any other
// interface carrying a marker a class, and any other named type a newtype.
// A constraint comes back from the message Validate writes, and a key from
// Key. What regenerating needs goes in a go target block. Under the //tdl:
// directives a `roundtrip` directive writes, it rebuilds the model the
// package was generated from. docs/design/go-backend.md has the mapping.
func (Backend) Import(_ context.Context, req *plugin.ImportRequest) (*plugin.ImportResponse, error) {
	r := newReader()
	file, err := r.read(req.GetFiles())
	if err != nil {
		return r.Response(nil, err)
	}
	return r.Response(reverse.Lower(file))
}

// annotation is the prefix of a //tdl: comment directive.
const annotation = "//tdl:"

// kv is one //tdl: directive.
type kv struct{ key, value string }

// role is what a type declaration reads as.
type role int

const (
	roleSkip role = iota
	roleStruct
	roleEnum   // a string type with constants
	roleSealed // an interface sealed by a marker, with a struct per variant
	roleClass
	roleNewtype
	roleAlias
	roleVariant // a sealed interface's variant struct
	roleKey     // the struct a Key method returns
)

type goFile struct {
	path    string
	file    *goast.File
	imports map[string]string // the name each import is referred to by, to its path
}

// typeInfo is one type declaration and what reading it found.
type typeInfo struct {
	name string
	file *goFile
	spec *goast.TypeSpec
	doc  *goast.CommentGroup
	role role

	methods  map[string]*goast.FuncDecl
	consts   []*goast.ValueSpec // a string enum's values
	variants []*typeInfo        // a sealed interface's variants
	owner    *typeInfo          // a variant's interface, or a key struct's entity
	anns     []kv
	// source is the declaration a //tdl:source directive carries.
	source ast.Decl
}

type reader struct {
	reverse.Reader
	fset  *token.FileSet
	files []*goFile
	pkg   string // the package clause

	types map[string]*typeInfo
	// order is every type declaration in reading order, and primary those
	// that read as a declaration of their own.
	order   []*typeInfo
	primary []*typeInfo

	// names maps each Go type read to its TDL name.
	names map[string]string
	// uses counts how often each type name is named in a type expression.
	uses map[string]int

	// foreign maps an import path and type name to the placeholder
	// declaration standing for it, and placeholders holds those in order.
	foreign      map[string]string
	placeholders []ast.Decl

	// decls is each primary type's declaration read, by Go name.
	decls map[string]ast.Decl
}

func newReader() *reader {
	return &reader{
		fset:    token.NewFileSet(),
		types:   map[string]*typeInfo{},
		names:   map[string]string{},
		uses:    map[string]int{},
		foreign: map[string]string{},
		decls:   map[string]ast.Decl{},
	}
}

func (r *reader) read(files []*plugin.File) (*ast.File, error) {
	if err := r.parse(files); err != nil {
		return nil, err
	}
	header, err := r.header()
	if err != nil {
		return nil, err
	}
	r.collect()
	if err := r.annotations(); err != nil {
		return nil, err
	}
	r.classify()
	r.nameAll()

	file := &ast.File{Filename: r.files[0].path}
	pkg := r.pkg
	if pkg == "main" {
		pkg = ""
	}
	if v, ok := get(header, "package"); ok {
		pkg = v
	}
	doc := r.packageDoc()
	if v, ok := get(header, "doc"); ok {
		doc = nil
		if v != "" {
			doc = strings.Split(v, "\n")
		}
	}
	if pkg != "" || len(doc) > 0 {
		file.Package = &ast.PackageDecl{Path: pkg, Doc: doc, DocP: make([]ast.Position, len(doc))}
	}
	for _, src := range all(header, "import") {
		imp, err := reverse.ParseImport(src)
		if err != nil {
			return nil, reverse.Failf(r.pos(token.NoPos), "//tdl:import %q is not one TDL import: %v", src, err)
		}
		file.Imports = append(file.Imports, imp)
	}
	var items []reverse.Item
	for _, v := range all(header, "item") {
		at, src, _ := strings.Cut(v, " ")
		n, err := strconv.Atoi(at)
		if err != nil {
			return nil, reverse.Failf(r.pos(token.NoPos), "a //tdl:item does not start with its index")
		}
		decl, err := reverse.ParseItem(src)
		if err != nil {
			return nil, reverse.Failf(r.pos(token.NoPos), "a //tdl:item is not TDL: %v", err)
		}
		items = append(items, reverse.Item{Decl: decl, At: n})
	}

	var emitted []reverse.Item
	for _, t := range r.primary {
		decl := r.decl(t)
		if decl == nil {
			continue
		}
		r.decls[t.name] = decl
		at := -1
		if v, ok := get(t.anns, "at"); ok {
			if at, err = strconv.Atoi(v); err != nil {
				return nil, reverse.Failf(r.pos(t.spec.Pos()), "the //tdl:at of %s is not an index", t.name)
			}
		}
		emitted = append(emitted, reverse.Item{Decl: decl, At: at})
	}
	for _, p := range r.placeholders {
		emitted = append(emitted, reverse.Item{Decl: p, At: -1})
	}
	file.Decls = reverse.Arrange(emitted, items)
	r.layout()
	if len(r.Entries) > 0 {
		file.Decls = append(file.Decls, &ast.TargetDecl{DeclHead: ast.DeclHead{N: Name}, For: pkg, Entries: r.Entries})
	}
	return file, nil
}

// parse parses every file, sorted by path, which must share one package
// clause.
func (r *reader) parse(files []*plugin.File) error {
	sorted := slices.SortedFunc(slices.Values(files), func(a, b *plugin.File) int { return cmp.Compare(a.GetPath(), b.GetPath()) })
	for _, f := range sorted {
		p := f.GetPath()
		switch {
		case strings.HasSuffix(p, "_test.go"):
			r.Warn(emit.LossUnsupported, &ir.Position{Filename: p}, "%s is a test file, which is not read", p)
			continue
		case !strings.HasSuffix(p, ".go"):
			return reverse.Failf(&ir.Position{Filename: p}, "import reads .go files, and was given %s", p)
		}
		parsed, err := goparser.ParseFile(r.fset, p, f.GetContent(), goparser.ParseComments|goparser.SkipObjectResolution)
		if err != nil {
			return reverse.Failf(&ir.Position{Filename: p}, "%s does not parse: %v", p, err)
		}
		if r.pkg == "" {
			r.pkg = parsed.Name.Name
		} else if parsed.Name.Name != r.pkg {
			return reverse.Failf(r.pos(parsed.Name.Pos()), "%s is package %s, and %s is package %s", p, parsed.Name.Name, r.files[0].path, r.pkg)
		}
		gf := &goFile{path: p, file: parsed, imports: map[string]string{}}
		for _, imp := range parsed.Imports {
			ipath, _ := strconv.Unquote(imp.Path.Value)
			name := defaultImportName(ipath)
			if imp.Name != nil {
				name = imp.Name.Name
			}
			gf.imports[name] = ipath
		}
		r.files = append(r.files, gf)
	}
	if len(r.files) == 0 {
		return reverse.Failf(nil, "import reads the .go files of one package, and was given none")
	}
	return nil
}

// defaultImportName is the name an import is referred to by without one:
// its last segment, past a major version and a go- prefix.
func defaultImportName(p string) string {
	p = major.ReplaceAllString(p, "")
	return strings.TrimPrefix(path.Base(p), "go-")
}

// header is the package-level annotations, from any file's package doc
// comment.
func (r *reader) header() ([]kv, error) {
	var out []kv
	for _, f := range r.files {
		anns, err := r.parseAnns(f.file.Doc)
		if err != nil {
			return nil, err
		}
		out = append(out, anns...)
	}
	return out, nil
}

// packageDoc is the package's doc comment: the first file's that has one.
func (r *reader) packageDoc() []string {
	var doc []string
	found := false
	for _, f := range r.files {
		lines, _ := readDoc(f.file.Doc, false)
		if len(lines) == 0 {
			continue
		}
		if found {
			r.Warn(emit.LossDoc, r.pos(f.file.Package), "%s's package doc comment is not read, since %s's comes first", f.path, r.files[0].path)
			continue
		}
		doc, found = lines, true
	}
	return doc
}

// parseAnns reads the //tdl: directives in a comment group. A value is a Go
// string literal, or an integer for at.
func (r *reader) parseAnns(cg *goast.CommentGroup) ([]kv, error) {
	if cg == nil {
		return nil, nil
	}
	var out []kv
	for _, c := range cg.List {
		rest, ok := strings.CutPrefix(c.Text, annotation)
		if !ok {
			continue
		}
		r.Roundtrip = true
		key, raw, _ := strings.Cut(rest, " ")
		value := raw
		if key != "at" {
			s, err := strconv.Unquote(raw)
			if err != nil {
				return nil, reverse.Failf(r.pos(c.Pos()), "%s%s does not hold a Go string", annotation, key)
			}
			value = s
		}
		out = append(out, kv{key, value})
	}
	return out, nil
}

// collect gathers every type declaration, constant, and method, and warns
// about what has no TDL form.
func (r *reader) collect() {
	for _, f := range r.files {
		for _, d := range f.file.Decls {
			gd, ok := d.(*goast.GenDecl)
			if !ok || gd.Tok != token.TYPE {
				continue
			}
			for _, s := range gd.Specs {
				spec := s.(*goast.TypeSpec)
				doc := spec.Doc
				if doc == nil && len(gd.Specs) == 1 {
					doc = gd.Doc
				}
				t := &typeInfo{name: spec.Name.Name, file: f, spec: spec, doc: doc, methods: map[string]*goast.FuncDecl{}}
				if _, dup := r.types[t.name]; dup {
					continue
				}
				r.types[t.name] = t
				r.order = append(r.order, t)
				r.count(spec.Type)
				if spec.TypeParams != nil {
					for _, p := range spec.TypeParams.List {
						r.count(p.Type)
					}
				}
			}
		}
	}

	for _, f := range r.files {
		for _, d := range f.file.Decls {
			switch d := d.(type) {
			case *goast.FuncDecl:
				r.method(d)
			case *goast.GenDecl:
				switch d.Tok {
				case token.CONST:
					r.constants(d)
				case token.VAR:
					r.vars(d)
				}
			}
		}
	}
}

// count counts the type names a type expression names.
func (r *reader) count(e goast.Expr) {
	goast.Inspect(e, func(n goast.Node) bool {
		if id, ok := n.(*goast.Ident); ok {
			r.uses[id.Name]++
		}
		return true
	})
}

func (r *reader) method(d *goast.FuncDecl) {
	if d.Recv == nil || len(d.Recv.List) != 1 {
		r.Warn(emit.LossUnsupported, r.pos(d.Pos()), "func %s has no TDL form", d.Name.Name)
		return
	}
	recv := receiverName(d.Recv.List[0].Type)
	t := r.types[recv]
	if t == nil {
		r.Warn(emit.LossUnsupported, r.pos(d.Pos()), "method %s.%s has no TDL form", recv, d.Name.Name)
		return
	}
	t.methods[d.Name.Name] = d
}

// receiverName is the type a receiver names, past a pointer and type
// arguments.
func receiverName(e goast.Expr) string {
	for {
		switch x := e.(type) {
		case *goast.StarExpr:
			e = x.X
		case *goast.ParenExpr:
			e = x.X
		case *goast.IndexExpr:
			e = x.X
		case *goast.IndexListExpr:
			e = x.X
		case *goast.Ident:
			return x.Name
		default:
			return ""
		}
	}
}

// constants attaches each string constant of a named type to it, as an
// enum's value.
func (r *reader) constants(d *goast.GenDecl) {
	for _, s := range d.Specs {
		spec := s.(*goast.ValueSpec)
		id, _ := spec.Type.(*goast.Ident)
		var lit *goast.BasicLit
		if len(spec.Values) == 1 {
			lit, _ = spec.Values[0].(*goast.BasicLit)
		}
		if id == nil || r.types[id.Name] == nil || len(spec.Names) != 1 || lit == nil || lit.Kind != token.STRING {
			r.Warn(emit.LossUnsupported, r.pos(spec.Pos()), "const %s has no TDL form", spec.Names[0].Name)
			continue
		}
		t := r.types[id.Name]
		if spec.Doc == nil && len(d.Specs) == 1 {
			spec.Doc = d.Doc
		}
		t.consts = append(t.consts, spec)
	}
}

// vars accepts the patterns Validate compiles.
func (r *reader) vars(d *goast.GenDecl) {
	for _, s := range d.Specs {
		spec := s.(*goast.ValueSpec)
		if len(spec.Names) == 1 && strings.HasPrefix(spec.Names[0].Name, "pattern") && len(spec.Values) == 1 {
			if call, ok := spec.Values[0].(*goast.CallExpr); ok && selector(call.Fun) == "regexp.MustCompile" {
				continue
			}
		}
		r.Warn(emit.LossUnsupported, r.pos(spec.Pos()), "var %s has no TDL form", spec.Names[0].Name)
	}
}

// selector is a qualified identifier as written, or "".
func selector(e goast.Expr) string {
	if s, ok := e.(*goast.SelectorExpr); ok {
		if x, ok := s.X.(*goast.Ident); ok {
			return x.Name + "." + s.Sel.Name
		}
	}
	return ""
}

// annotations reads each type's //tdl: directives.
func (r *reader) annotations() error {
	for _, t := range r.order {
		anns, err := r.parseAnns(t.doc)
		if err != nil {
			return err
		}
		t.anns = anns
		if src, ok := get(anns, "source"); ok {
			decl, err := reverse.ParseItem(src)
			if err != nil {
				return reverse.Failf(r.pos(t.spec.Pos()), "the //tdl:source of %s is not TDL: %v", t.name, err)
			}
			t.source = decl
		}
	}
	return nil
}

// classify decides what each type declaration reads as.
func (r *reader) classify() {
	for _, t := range r.order {
		switch t.spec.Type.(type) {
		case *goast.InterfaceType:
			if t.spec.Assign.IsValid() {
				t.role = roleAlias
			} else if r.marker(t) == nil {
				t.role = roleSkip
			} else {
				t.role = roleClass
			}
		case *goast.StructType:
			t.role = roleStruct
			if t.spec.Assign.IsValid() {
				t.role = roleAlias
			}
		default:
			switch {
			case t.spec.Assign.IsValid():
				t.role = roleAlias
			case len(t.consts) > 0 || (isIdent(t.spec.Type, "string") && t.source != nil && isEnum(t.source)):
				t.role = roleEnum
			default:
				t.role = roleNewtype
			}
		}
	}

	// A Key method returning a struct named for the entity makes that
	// struct its key.
	for _, t := range r.order {
		m := t.methods["Key"]
		if t.role != roleStruct || m == nil || m.Type.Results == nil || len(m.Type.Results.List) != 1 {
			continue
		}
		if k := r.types[receiverName(m.Type.Results.List[0].Type)]; k != nil && k != t && k.role == roleStruct && k.name == t.name+"Key" {
			k.role, k.owner = roleKey, t
		}
	}

	// An interface is sealed when every type carrying its marker is a
	// struct named after it that nothing else names.
	for _, t := range r.order {
		if t.role != roleClass {
			continue
		}
		var impls []*typeInfo
		for _, o := range r.order {
			if _, ok := o.methods["is"+t.name]; ok && o != t {
				impls = append(impls, o)
			}
		}
		// An annotation says which an interface is, and then every struct
		// carrying its marker is a variant, whatever its name.
		sealed := len(impls) > 0
		switch {
		case t.source != nil:
			sealed = isEnum(t.source)
		default:
			for _, o := range impls {
				sealed = sealed && o.role == roleStruct && o.source == nil && len(o.name) > len(t.name) && strings.HasPrefix(o.name, t.name) && r.uses[o.name] == 0
			}
		}
		if !sealed {
			if t.spec.TypeParams != nil || len(r.marker(t).Type.(*goast.FuncType).Params.List) > 0 {
				t.role = roleSkip
			}
			continue
		}
		t.role = roleSealed
		for _, o := range impls {
			o.role, o.owner = roleVariant, t
			t.variants = append(t.variants, o)
		}
	}

	for _, t := range r.order {
		switch t.role {
		case roleVariant, roleKey:
		case roleSkip:
			if t.source == nil {
				r.Warn(emit.LossUnsupported, r.pos(t.spec.Pos()), "%s is an interface without a method marking it, which TDL has no form for", t.name)
				continue
			}
			r.primary = append(r.primary, t)
		default:
			r.primary = append(r.primary, t)
		}
	}
}

func isIdent(e goast.Expr, name string) bool {
	id, ok := e.(*goast.Ident)
	return ok && id.Name == name
}

func isEnum(d ast.Decl) bool {
	_, ok := d.(*ast.EnumDecl)
	return ok
}

// marker is an interface's marker method, is<Name>, or nil.
func (r *reader) marker(t *typeInfo) *goast.Field {
	it, ok := t.spec.Type.(*goast.InterfaceType)
	if !ok {
		return nil
	}
	for _, m := range it.Methods.List {
		if len(m.Names) == 1 && m.Names[0].Name == "is"+t.name {
			return m
		}
	}
	return nil
}

// nameAll gives every type read its TDL name before any is read, since a
// field may name one declared after it.
func (r *reader) nameAll() {
	for _, t := range r.primary {
		name := t.name
		if t.source != nil {
			name = t.source.Name()
		}
		r.names[t.name] = name
	}
}

// decl reads one type declaration, or returns nil for one it skips.
func (r *reader) decl(t *typeInfo) ast.Decl {
	if t.source != nil {
		return t.source
	}
	name := r.names[t.name]
	if !token.IsExported(t.name) {
		r.Warn(emit.LossName, r.pos(t.spec.Pos()), "%s is unexported, and regenerates as %s", t.name, exported(t.name))
	}
	head := r.head(name, t.doc, t.spec.Pos())
	params, requires := r.params(t, name)
	scope := paramSet(params)

	switch t.role {
	case roleStruct:
		d := &ast.StructDecl{DeclHead: head, Keyword: "type", Params: params, Requires: requires}
		r.checkMethods(t, "")
		if keys := r.key(t, name); len(keys) > 0 {
			d.Conforms = append(d.Conforms, &ast.ClassRef{N: "Entity"})
		}
		d.Conforms = append(d.Conforms, r.markers(t, "")...)
		cs := r.validate(t)
		for _, f := range r.fields(t.spec.Type.(*goast.StructType).Fields, name, scope, cs) {
			d.Members = append(d.Members, f)
		}
		if len(d.Members) > 0 {
			d.End = d.P
		}
		return d
	case roleEnum:
		return r.enum(t, head)
	case roleSealed:
		return r.sealed(t, head, params, requires, scope)
	case roleClass:
		d := &ast.ClassDecl{DeclHead: head}
		for _, m := range t.spec.Type.(*goast.InterfaceType).Methods.List {
			if len(m.Names) > 0 {
				continue
			}
			if ref := r.classRef(m.Type, name); ref != nil {
				d.Conforms = append(d.Conforms, ref)
			}
		}
		for _, m := range t.spec.Type.(*goast.InterfaceType).Methods.List {
			if len(m.Names) == 1 && m.Names[0].Name != "is"+t.name {
				r.Warn(emit.LossUnsupported, r.pos(m.Pos()), "%s's method %s has no TDL form", t.name, m.Names[0].Name)
			}
		}
		return d
	case roleNewtype:
		base := r.typeRef(t.spec.Type, scope, name)
		if base == nil {
			return nil
		}
		r.checkMethods(t, "")
		if ms := r.markers(t, ""); len(ms) > 0 {
			r.Warn(emit.LossClass, r.pos(t.spec.Pos()), "%s carries the marker of %s, and a newtype states no conformance", t.name, ms[0].N)
		}
		d := &ast.NewtypeDecl{DeclHead: head, Params: params, Base: base, Requires: requires}
		d.Constraints = r.ownConstraints(t, r.validate(t)[""])
		if len(d.Constraints) > 0 {
			d.End = d.P
		}
		return d
	case roleAlias:
		target := r.typeRef(t.spec.Type, scope, name)
		if target == nil {
			return nil
		}
		return &ast.AliasDecl{DeclHead: head, Params: params, Target: target}
	}
	return nil
}

// head is a node's head from its doc comment.
func (r *reader) head(name string, doc *goast.CommentGroup, p token.Pos) ast.DeclHead {
	lines, dep := readDoc(doc, true)
	h := reverse.Head(name, lines, r.at(p), false)
	h.Dep = dep
	return h
}

// readDoc is the doc comment and deprecation a comment group holds: its
// text without directives, and with deprecated set, a last paragraph
// beginning "Deprecated: ", whose lines are the reason's. The reason
// Generate writes for none reads as none.
func readDoc(cg *goast.CommentGroup, deprecated bool) ([]string, *ast.Deprecation) {
	text := strings.TrimSuffix(cg.Text(), "\n")
	if text == "" {
		return nil, nil
	}
	lines := strings.Split(text, "\n")
	if !deprecated {
		return lines, nil
	}
	start := len(lines) - 1
	for start > 0 && lines[start-1] != "" {
		start--
	}
	reason, ok := strings.CutPrefix(lines[start], "Deprecated: ")
	if !ok {
		return lines, nil
	}
	reason = strings.Join(append([]string{reason}, lines[start+1:]...), "\n")
	if reason == defaultReason {
		reason = ""
	}
	lines = lines[:start]
	if n := len(lines); n > 0 && lines[n-1] == "" {
		lines = lines[:n-1]
	}
	return lines, &ast.Deprecation{Reason: reason}
}

// defaultReason is the deprecation Generate writes for one without a
// reason.
const defaultReason = "this declaration is on its way out."

// params reads a type's parameters and the classes constraining them. A
// comparable constraint is inferred again generating.
func (r *reader) params(t *typeInfo, name string) ([]*ast.TypeParam, []*ast.ClassRef) {
	if t.spec.TypeParams == nil {
		return nil, nil
	}
	var params []*ast.TypeParam
	var requires []*ast.ClassRef
	for _, f := range t.spec.TypeParams.List {
		for _, n := range f.Names {
			params = append(params, &ast.TypeParam{P: r.at(n.Pos()), N: n.Name})
			for _, c := range r.constraintClasses(f.Type, name, n.Name) {
				requires = append(requires, &ast.ClassRef{N: c, Args: []*ast.TypeArg{{Type: &ast.TypeRef{N: n.Name}}}})
			}
		}
	}
	return params, requires
}

// constraintClasses is the classes a type parameter's constraint names.
func (r *reader) constraintClasses(e goast.Expr, owner, param string) []string {
	var out []string
	var visit func(goast.Expr)
	visit = func(e goast.Expr) {
		switch x := e.(type) {
		case *goast.Ident:
			switch {
			case x.Name == "any" || x.Name == "comparable":
			case r.types[x.Name] != nil && r.types[x.Name].role == roleClass:
				out = append(out, r.names[x.Name])
			default:
				r.Warn(emit.LossGeneric, r.pos(x.Pos()), "%s's parameter %s is constrained by %s, which TDL has no form for", owner, param, x.Name)
			}
		case *goast.InterfaceType:
			for _, m := range x.Methods.List {
				if len(m.Names) > 0 {
					r.Warn(emit.LossGeneric, r.pos(m.Pos()), "%s's parameter %s is constrained by a method, which TDL has no form for", owner, param)
					continue
				}
				visit(m.Type)
			}
		default:
			r.Warn(emit.LossGeneric, r.pos(e.Pos()), "%s's parameter %s has a constraint TDL has no form for", owner, param)
		}
	}
	visit(e)
	return out
}

func paramSet(params []*ast.TypeParam) map[string]bool {
	out := map[string]bool{}
	for _, p := range params {
		out[p.N] = true
	}
	return out
}

// markers is the classes whose markers a type carries, other than the
// interface sealing it.
func (r *reader) markers(t *typeInfo, sealedBy string) []*ast.ClassRef {
	var out []*ast.ClassRef
	for _, m := range sortedMethods(t) {
		class, ok := strings.CutPrefix(m.Name.Name, "is")
		if !ok || class == sealedBy {
			continue
		}
		if c := r.types[class]; c != nil && c.role == roleClass {
			out = append(out, &ast.ClassRef{N: r.names[class]})
		}
	}
	return out
}

// sortedMethods is a type's methods in source order.
func sortedMethods(t *typeInfo) []*goast.FuncDecl {
	var out []*goast.FuncDecl
	for _, m := range t.methods {
		out = append(out, m)
	}
	slices.SortFunc(out, func(a, b *goast.FuncDecl) int { return cmp.Compare(a.Pos(), b.Pos()) })
	return out
}

// checkMethods warns about each method of a type Generate does not
// write.
func (r *reader) checkMethods(t *typeInfo, sealedBy string) {
	for _, m := range sortedMethods(t) {
		name := m.Name.Name
		switch {
		case name == "Validate" || name == "validate":
		case name == "Key" && t.role == roleStruct:
		case name == "is"+sealedBy && sealedBy != "":
		case strings.HasPrefix(name, "is") && r.types[name[2:]] != nil && r.types[name[2:]].role == roleClass:
		default:
			r.Warn(emit.LossUnsupported, r.pos(m.Pos()), "method %s.%s has no TDL form", t.name, name)
		}
	}
}

// key reads an entity's Key method: the field it returns, or the fields of
// the key struct it returns. It writes the key directive.
func (r *reader) key(t *typeInfo, name string) []string {
	m := t.methods["Key"]
	if m == nil || m.Body == nil || len(m.Body.List) != 1 {
		return nil
	}
	ret, ok := m.Body.List[0].(*goast.ReturnStmt)
	if !ok || len(ret.Results) != 1 {
		return nil
	}
	var goFields []string
	switch x := ret.Results[0].(type) {
	case *goast.SelectorExpr:
		goFields = []string{x.Sel.Name}
	case *goast.CompositeLit:
		for _, e := range x.Elts {
			kve, ok := e.(*goast.KeyValueExpr)
			if !ok {
				return nil
			}
			sel, ok := kve.Value.(*goast.SelectorExpr)
			if !ok {
				return nil
			}
			goFields = append(goFields, sel.Sel.Name)
		}
	default:
		return nil
	}
	var names []string
	var lits []*ast.Literal
	for _, f := range goFields {
		n := unexport(f)
		names = append(names, n)
		lits = append(lits, &ast.Literal{Kind: ast.LitName, Text: n})
	}
	r.Directive(name, "key", lits...)
	return names
}

// fields reads a struct's fields. cs holds the constraints Validate checks,
// by field name.
func (r *reader) fields(list *goast.FieldList, owner string, scope map[string]bool, cs map[string][]*ast.Constraint) []*ast.Field {
	var out []*ast.Field
	read := map[string]bool{}
	for _, f := range list.List {
		if len(f.Names) == 0 {
			r.Warn(emit.LossUnsupported, r.pos(f.Pos()), "%s embeds %s, which TDL has no form for", owner, exprString(f.Type))
			continue
		}
		for _, n := range f.Names {
			if n.Name == "_" {
				r.Warn(emit.LossUnsupported, r.pos(n.Pos()), "%s has a blank field, which TDL has no form for", owner)
				continue
			}
			name := unexport(n.Name)
			path := owner + "." + name
			typ := r.typeRef(f.Type, scope, path)
			if typ == nil {
				continue
			}
			switch {
			case !token.IsExported(n.Name):
				r.Warn(emit.LossName, r.pos(n.Pos()), "%s.%s is unexported, and regenerates as %s", owner, n.Name, exported(n.Name))
			case exported(name) != n.Name:
				r.Directive(path, "name", reverse.StrLit(n.Name))
			}
			if f.Tag != nil {
				if tag, err := strconv.Unquote(f.Tag.Value); err == nil {
					r.Directive(path, "tag", reverse.StrLit(tag))
				}
			}
			read[name] = true
			field := &ast.Field{DeclHead: r.head(name, f.Doc, n.Pos()), Type: typ, Constraints: cs[name]}
			if len(field.Constraints) > 0 {
				field.End = field.P
			}
			out = append(out, field)
		}
	}
	for _, name := range slices.Sorted(maps.Keys(cs)) {
		if !read[name] {
			r.Warn(emit.LossConstraint, r.pos(list.Pos()), "%s's validate checks %q, which names none of its fields", owner, name)
		}
	}
	return out
}

// unexport is the TDL name a Go identifier reads as: its leading capitals
// lower case, but for one starting the next word, so ID is id, Last4 is
// last4, and HTTPServer is httpServer.
func unexport(name string) string {
	r := []rune(name)
	n := 0
	for n < len(r) && unicode.IsUpper(r[n]) {
		n++
	}
	if n > 1 && n < len(r) && unicode.IsLower(r[n]) {
		n--
	}
	for i := range n {
		r[i] = unicode.ToLower(r[i])
	}
	return string(r)
}

// validate reads the constraints a type's validate method checks, from
// the messages it writes: "%s.<field>: <constraint>: <detail>", or
// "%s: <constraint>: <detail>" for the value itself, keyed "".
func (r *reader) validate(t *typeInfo) map[string][]*ast.Constraint {
	m := t.methods["validate"]
	if m == nil || m.Body == nil {
		return nil
	}
	out := map[string][]*ast.Constraint{}
	goast.Inspect(m.Body, func(n goast.Node) bool {
		call, ok := n.(*goast.CallExpr)
		if !ok || selector(call.Fun) != "fmt.Errorf" || len(call.Args) == 0 {
			return true
		}
		lit, ok := call.Args[0].(*goast.BasicLit)
		if !ok || lit.Kind != token.STRING {
			return true
		}
		msg, err := strconv.Unquote(lit.Value)
		if err != nil {
			return true
		}
		field, c, ok := parseCheck(msg)
		if !ok {
			r.Warn(emit.LossConstraint, r.pos(lit.Pos()), "%s's check %q is not one import reads", t.name, msg)
			return true
		}
		out[field] = append(out[field], c)
		return true
	})
	return out
}

// parseCheck reads one message validate writes into the field it is about
// and the constraint as TDL.
func parseCheck(msg string) (string, *ast.Constraint, bool) {
	rest, ok := strings.CutPrefix(msg, "%s")
	if !ok {
		return "", nil, false
	}
	field := ""
	if after, ok := strings.CutPrefix(rest, "."); ok {
		name, tail, ok := strings.Cut(after, ": ")
		if !ok {
			return "", nil, false
		}
		field, rest = name, ": "+tail
	}
	rest, ok = strings.CutPrefix(rest, ": ")
	i := strings.LastIndex(rest, ": ")
	if !ok || i < 0 {
		return "", nil, false
	}
	text := strings.ReplaceAll(rest[:i], "%%", "%")
	f, err := reverse.ParseField("x: string where { " + text + " }")
	if err != nil || len(f.Constraints) != 1 {
		return "", nil, false
	}
	return strings.ReplaceAll(field, "%%", "%"), f.Constraints[0], true
}

// ownConstraints drops from a newtype's constraints those it inherits from
// a newtype it is over, which Validate checks again and lowering adds back.
func (r *reader) ownConstraints(t *typeInfo, cs []*ast.Constraint) []*ast.Constraint {
	id, ok := t.spec.Type.(*goast.Ident)
	if !ok {
		return cs
	}
	base := r.types[id.Name]
	if base == nil || base.role != roleNewtype {
		return cs
	}
	inherited := r.validate(base)[""]
	if len(inherited) > len(cs) {
		return cs
	}
	own := len(cs) - len(inherited)
	for i, c := range inherited {
		if constraintText(c) != constraintText(cs[own+i]) {
			return cs
		}
	}
	return cs[:own]
}

func constraintText(c *ast.Constraint) string {
	return ast.PrintField(&ast.Field{DeclHead: ast.DeclHead{N: "x"}, Type: &ast.TypeRef{N: "T"}, Constraints: []*ast.Constraint{c}})
}

// enum reads a string type with constants as an enum: each constant's
// value is a variant's name.
func (r *reader) enum(t *typeInfo, head ast.DeclHead) ast.Decl {
	d := &ast.EnumDecl{DeclHead: head, Conforms: r.markers(t, "")}
	r.checkMethods(t, "")
	if !isIdent(t.spec.Type, "string") {
		r.Warn(emit.LossUnsupported, r.pos(t.spec.Pos()), "%s's constants are not strings, and regenerate as one", t.name)
	}
	for _, c := range t.consts {
		value, _ := strconv.Unquote(c.Values[0].(*goast.BasicLit).Value)
		if !token.IsIdentifier(value) {
			r.Warn(emit.LossUnsupported, r.pos(c.Pos()), "%s's value %q is not a TDL name", c.Names[0].Name, value)
			continue
		}
		if ident := c.Names[0].Name; ident != t.name+exported(value) {
			r.Directive(head.N+"."+value, "name", reverse.StrLit(ident))
		}
		d.Variants = append(d.Variants, &ast.Variant{DeclHead: r.head(value, c.Doc, c.Pos())})
	}
	if len(d.Variants) > 0 {
		d.End = d.P
	}
	return d
}

// sealed reads a sealed interface as an enum with a variant per struct
// carrying its marker.
func (r *reader) sealed(t *typeInfo, head ast.DeclHead, params []*ast.TypeParam, requires []*ast.ClassRef, scope map[string]bool) ast.Decl {
	d := &ast.EnumDecl{DeclHead: head, Params: params, Requires: requires}
	for _, m := range t.spec.Type.(*goast.InterfaceType).Methods.List {
		if len(m.Names) == 0 {
			if ref := r.classRef(m.Type, t.name); ref != nil {
				d.Conforms = append(d.Conforms, ref)
			}
		}
	}
	for _, v := range t.variants {
		vname := strings.TrimPrefix(v.name, t.name)
		path := head.N + "." + vname
		r.checkMethods(v, t.name)
		variant := &ast.Variant{DeclHead: r.head(vname, v.doc, v.spec.Pos())}
		variant.Fields = r.fields(v.spec.Type.(*goast.StructType).Fields, path, scope, r.validate(v))
		if len(variant.Fields) > 0 {
			variant.End = variant.P
		}
		d.Variants = append(d.Variants, variant)
	}
	if len(d.Variants) > 0 {
		d.End = d.P
	}
	return d
}

// classRef reads an embedded interface naming a class.
func (r *reader) classRef(e goast.Expr, owner string) *ast.ClassRef {
	id, ok := e.(*goast.Ident)
	if ok && r.types[id.Name] != nil && r.types[id.Name].role == roleClass {
		return &ast.ClassRef{P: r.at(id.Pos()), N: r.names[id.Name]}
	}
	r.Warn(emit.LossUnsupported, r.pos(e.Pos()), "%s embeds %s, which is not a class", owner, exprString(e))
	return nil
}

// builtins maps a predeclared Go type to the primitive it reads as, and
// whether that regenerates as written.
var builtins = map[string]struct {
	prim  string
	exact bool
}{
	"string":  {"string", true},
	"bool":    {"bool", true},
	"int64":   {"int64", true},
	"int32":   {"int32", true},
	"uint32":  {"uint32", true},
	"uint64":  {"uint64", true},
	"float32": {"float32", true},
	"float64": {"float64", true},
	"int":     {"int64", false},
	"int8":    {"int32", false},
	"int16":   {"int32", false},
	"rune":    {"int32", false},
	"uint":    {"uint64", false},
	"uint8":   {"uint32", false},
	"byte":    {"uint32", false},
	"uint16":  {"uint32", false},
	"uintptr": {"uint64", false},
}

// typeRef reads a type expression, or warns and returns nil. scope holds
// the type parameters in scope.
func (r *reader) typeRef(e goast.Expr, scope map[string]bool, what string) *ast.TypeRef {
	unsupported := func(format string, args ...any) *ast.TypeRef {
		r.Warn(emit.LossUnsupported, r.pos(e.Pos()), "%s is %s, which TDL has no form for", what, fmt.Sprintf(format, args...))
		return nil
	}
	p := r.at(e.Pos())
	switch x := e.(type) {
	case *goast.ParenExpr:
		return r.typeRef(x.X, scope, what)
	case *goast.Ident:
		switch t := r.types[x.Name]; {
		case scope[x.Name]:
			return &ast.TypeRef{P: p, N: x.Name}
		case t != nil:
			return r.named(t, nil, x.Pos(), what)
		}
		b, ok := builtins[x.Name]
		if !ok {
			return unsupported("%s", x.Name)
		}
		if !b.exact {
			r.Warn(emit.LossPrimitive, r.pos(x.Pos()), "%s is %s, which reads as %s and regenerates as %s", what, x.Name, b.prim, primitives[b.prim])
		}
		return &ast.TypeRef{P: p, N: b.prim}
	case *goast.SelectorExpr:
		return r.qualified(x, what)
	case *goast.StarExpr:
		inner := r.typeRef(x.X, scope, what)
		if inner == nil {
			return nil
		}
		if inner.Optional {
			return unsupported("a pointer to a pointer")
		}
		inner.Optional = true
		return inner
	case *goast.ArrayType:
		if x.Len != nil {
			return unsupported("an array")
		}
		if isIdent(x.Elt, "byte") || isIdent(x.Elt, "uint8") {
			return &ast.TypeRef{P: p, N: "bytes"}
		}
		elem := r.typeRef(x.Elt, scope, what)
		if elem == nil {
			return nil
		}
		return &ast.TypeRef{P: p, List: elem}
	case *goast.MapType:
		key := r.typeRef(x.Key, scope, what)
		if key == nil {
			return nil
		}
		if s, ok := x.Value.(*goast.StructType); ok && len(s.Fields.List) == 0 {
			return &ast.TypeRef{P: p, Set: key}
		}
		value := r.typeRef(x.Value, scope, what)
		if value == nil {
			return nil
		}
		return &ast.TypeRef{P: p, MapKey: key, MapValue: value}
	case *goast.IndexExpr, *goast.IndexListExpr:
		base, args := indexed(x)
		id, ok := base.(*goast.Ident)
		if !ok || r.types[id.Name] == nil {
			return unsupported("%s", exprString(e))
		}
		var targs []*ast.TypeArg
		for _, a := range args {
			ref := r.typeRef(a, scope, what)
			if ref == nil {
				return nil
			}
			targs = append(targs, &ast.TypeArg{P: ref.P, Type: ref})
		}
		return r.named(r.types[id.Name], targs, e.Pos(), what)
	case *goast.StructType:
		return unsupported("an anonymous struct")
	case *goast.InterfaceType:
		return unsupported("an interface")
	case *goast.FuncType:
		return unsupported("a func")
	case *goast.ChanType:
		return unsupported("a channel")
	}
	return unsupported("%s", exprString(e))
}

// named reads a reference to a type the package declares.
func (r *reader) named(t *typeInfo, args []*ast.TypeArg, p token.Pos, what string) *ast.TypeRef {
	switch t.role {
	case roleClass:
		r.Warn(emit.LossUnsupported, r.pos(p), "%s names the interface %s, which TDL has no form for", what, t.name)
		return nil
	case roleSkip, roleVariant, roleKey:
		if t.source == nil {
			r.Warn(emit.LossUnsupported, r.pos(p), "%s names %s, which is not imported", what, t.name)
			return nil
		}
	}
	return &ast.TypeRef{P: r.at(p), N: r.names[t.name], Args: args}
}

// qualified reads a type another package declares: time's, or a foreign
// type, which reads as a placeholder declaration mapped to it.
func (r *reader) qualified(x *goast.SelectorExpr, what string) *ast.TypeRef {
	p := r.at(x.Pos())
	pkg, _ := x.X.(*goast.Ident)
	ipath := ""
	if pkg != nil {
		ipath = r.fileOf(x.Pos()).imports[pkg.Name]
	}
	switch {
	case ipath == "time" && x.Sel.Name == "Time":
		return &ast.TypeRef{P: p, N: "instant"}
	case ipath == "time" && x.Sel.Name == "Duration":
		return &ast.TypeRef{P: p, N: "duration"}
	case ipath == "" || !token.IsExported(x.Sel.Name):
		r.Warn(emit.LossUnsupported, r.pos(x.Pos()), "%s is %s, which is not imported", what, exprString(x))
		return nil
	}
	key := ipath + "." + x.Sel.Name
	if name, ok := r.foreign[key]; ok {
		return &ast.TypeRef{P: p, N: name}
	}
	name := x.Sel.Name
	if r.Roundtrip {
		return &ast.TypeRef{P: p, N: name}
	}
	if _, taken := r.types[name]; taken || slices.Contains(slices.Collect(mapValues(r.foreign)), name) {
		r.Warn(emit.LossUnsupported, r.pos(x.Pos()), "%s is %s, whose name the package already takes", what, exprString(x))
		return nil
	}
	r.foreign[key] = name
	r.placeholders = append(r.placeholders, &ast.StructDecl{DeclHead: ast.DeclHead{N: name, P: p}, Keyword: "type"})
	r.Directive(name, "foreign", reverse.StrLit(ipath), reverse.StrLit(x.Sel.Name))
	return &ast.TypeRef{P: p, N: name}
}

func mapValues(m map[string]string) func(func(string) bool) {
	return func(yield func(string) bool) {
		for _, v := range m {
			if !yield(v) {
				return
			}
		}
	}
}

func indexed(e goast.Expr) (goast.Expr, []goast.Expr) {
	switch x := e.(type) {
	case *goast.IndexExpr:
		return x.X, []goast.Expr{x.Index}
	case *goast.IndexListExpr:
		return x.X, x.Indices
	}
	return e, nil
}

// layout writes a file directive when the files are not one per
// declaration, named for it, and warns when they are several.
func (r *reader) layout() {
	if r.Roundtrip {
		return
	}
	perDecl := true
	held := map[string]int{}
	for _, t := range r.primary {
		held[t.file.path]++
		perDecl = perDecl && path.Base(t.file.path) == fileName(r.names[t.name])
	}
	for _, f := range r.files {
		perDecl = perDecl && held[f.path] == 1
	}
	switch {
	case perDecl:
	case len(r.files) == 1:
		r.Directive("", "file", reverse.StrLit(path.Base(r.files[0].path)))
	default:
		r.Warn(emit.LossUnsupported, nil, "the package's files regenerate as one file per declaration")
	}
}

func (r *reader) fileOf(p token.Pos) *goFile {
	name := r.fset.Position(p).Filename
	for _, f := range r.files {
		if f.path == name {
			return f
		}
	}
	return r.files[0]
}

func (r *reader) pos(p token.Pos) *ir.Position {
	if !p.IsValid() {
		if len(r.files) > 0 {
			return &ir.Position{Filename: r.files[0].path}
		}
		return nil
	}
	at := r.fset.Position(p)
	return &ir.Position{Filename: at.Filename, Line: int32(at.Line), Column: int32(at.Column)}
}

func (r *reader) at(p token.Pos) ast.Position {
	if !p.IsValid() {
		return ast.Position{}
	}
	at := r.fset.Position(p)
	return ast.Position{Filename: at.Filename, Line: at.Line, Col: at.Column}
}

// exprString is a Go expression as source.
func exprString(e goast.Expr) string {
	var b bytes.Buffer
	if err := format.Node(&b, token.NewFileSet(), e); err != nil {
		return "?"
	}
	return b.String()
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

// Normalize puts Go files into the normal form the round-trip corpus
// compares: each file through go/format, keeping doc comments only.
func Normalize(files []*plugin.File) ([]*plugin.File, error) {
	out := make([]*plugin.File, len(files))
	for i, f := range files {
		fset := token.NewFileSet()
		parsed, err := goparser.ParseFile(fset, f.GetPath(), f.GetContent(), goparser.ParseComments|goparser.SkipObjectResolution)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", f.GetPath(), err)
		}
		docs := map[*goast.CommentGroup]bool{parsed.Doc: true}
		goast.Inspect(parsed, func(n goast.Node) bool {
			switch x := n.(type) {
			case *goast.GenDecl:
				docs[x.Doc] = true
			case *goast.FuncDecl:
				docs[x.Doc] = true
			case *goast.TypeSpec:
				docs[x.Doc] = true
			case *goast.ValueSpec:
				docs[x.Doc] = true
			case *goast.Field:
				docs[x.Doc] = true
			}
			return true
		})
		parsed.Comments = slices.DeleteFunc(parsed.Comments, func(cg *goast.CommentGroup) bool { return !docs[cg] })
		var b bytes.Buffer
		if err := format.Node(&b, fset, parsed); err != nil {
			return nil, fmt.Errorf("%s: %w", f.GetPath(), err)
		}
		out[i] = &plugin.File{Path: f.GetPath(), Content: b.Bytes()}
	}
	return out, nil
}
