package typescript

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/unstoppablemango/tdl/ast"
	"github.com/unstoppablemango/tdl/backend/internal/emit"
	"github.com/unstoppablemango/tdl/backend/internal/reverse"
	"github.com/unstoppablemango/tdl/ir"
	"github.com/unstoppablemango/tdl/lex"
	"github.com/unstoppablemango/tdl/plugin"
)

// outlineScript reads TypeScript with the compiler API, under node.
//
//go:embed outline.js
var outlineScript string

// defaultFile names the file of a model without a package.
const defaultFile = "model"

// Import reads one .ts file into a model: each interface a struct, each
// union of string literals an enum, a union of interfaces that nothing
// else names, each tagged by one string-literal property, as Generate
// writes a fielded enum, an enum with fields, and any other type alias a
// newtype. The file's name is the package. What regenerating needs goes in
// a typescript target block. Under the JSDoc @tdl tags a `roundtrip`
// directive writes, it rebuilds the model the file was generated from.
//
// The file is parsed by the TypeScript compiler, which import runs under
// node, so both must be on PATH. docs/design/schema-backends.md has the
// mapping.
func (Backend) Import(ctx context.Context, req *plugin.ImportRequest) (*plugin.ImportResponse, error) {
	r := &reader{names: map[string]string{}, sums: map[string]*sum{}, variants: map[string]bool{}}
	return r.Response(r.read(ctx, req.GetFiles()))
}

// The outline the script writes.
type (
	outline struct {
		Files []outlineFile `json:"files"`
	}
	outlineFile struct {
		Path        string `json:"path"`
		Diagnostics []struct {
			Line    int    `json:"line"`
			Col     int    `json:"col"`
			Message string `json:"message"`
		} `json:"diagnostics,omitempty"`
		Statements []*statement `json:"statements"`
	}
	statement struct {
		Kind     string    `json:"kind"`
		Syntax   string    `json:"syntax"`
		Name     string    `json:"name,omitempty"`
		Line     int       `json:"line,omitempty"`
		Col      int       `json:"col,omitempty"`
		Docs     []string  `json:"docs,omitempty"`
		Exported bool      `json:"exported,omitempty"`
		Declare  bool      `json:"declare,omitempty"`
		Params   int       `json:"params,omitempty"`
		Heritage bool      `json:"heritage,omitempty"`
		Members  []*member `json:"members,omitempty"`
		Type     *tsType   `json:"type,omitempty"`
	}
	member struct {
		Kind     string   `json:"kind"`
		Syntax   string   `json:"syntax,omitempty"`
		Name     string   `json:"name,omitempty"`
		Line     int      `json:"line,omitempty"`
		Col      int      `json:"col,omitempty"`
		Docs     []string `json:"docs,omitempty"`
		Quoted   bool     `json:"quoted,omitempty"`
		Optional bool     `json:"optional,omitempty"`
		Readonly bool     `json:"readonly,omitempty"`
		Type     *tsType  `json:"type,omitempty"`
		Init     *tsLit   `json:"init,omitempty"`
	}
	tsType struct {
		K     string    `json:"k"`
		Name  string    `json:"name,omitempty"`
		Args  []*tsType `json:"args,omitempty"`
		Elem  *tsType   `json:"elem,omitempty"`
		Types []*tsType `json:"types,omitempty"`
		Text  string    `json:"text,omitempty"`
		tsLit
	}
	tsLit struct {
		Str  *string `json:"str,omitempty"`
		Num  *string `json:"num,omitempty"`
		Bool *bool   `json:"bool,omitempty"`
	}
)

// ErrNoCompiler is why import cannot run: node or the TypeScript compiler
// is not on PATH.
var ErrNoCompiler = errors.New("importing TypeScript runs the TypeScript compiler under node, and needs node and tsc on PATH")

// Compiler finds node and the TypeScript package's directory, from tsc on
// PATH, or returns [ErrNoCompiler].
func Compiler() (node, pkg string, err error) {
	node, err = exec.LookPath("node")
	if err != nil {
		return "", "", ErrNoCompiler
	}
	tsc, err := exec.LookPath("tsc")
	if err != nil {
		return "", "", ErrNoCompiler
	}
	real, err := filepath.EvalSymlinks(tsc)
	if err != nil {
		return "", "", ErrNoCompiler
	}
	// tsc is the package's bin/tsc, or a wrapper in a prefix's bin, as
	// nixpkgs installs it, that runs lib/node_modules/typescript/bin/tsc.
	prefix := filepath.Dir(filepath.Dir(real))
	for _, dir := range []string{prefix, filepath.Join(prefix, "lib", "node_modules", "typescript")} {
		if _, err := os.Stat(filepath.Join(dir, "package.json")); err == nil {
			return node, dir, nil
		}
	}
	return "", "", ErrNoCompiler
}

// parse runs the compiler over files and returns their outline.
func parse(ctx context.Context, files []*plugin.File) (*outline, error) {
	node, pkg, err := Compiler()
	if err != nil {
		return nil, err
	}
	type input struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	in := make([]input, len(files))
	for i, f := range files {
		in[i] = input{f.GetPath(), string(f.GetContent())}
	}
	data, err := json.Marshal(in)
	if err != nil {
		return nil, err
	}
	cmd := exec.CommandContext(ctx, node, "-e", outlineScript, pkg)
	cmd.Stdin = bytes.NewReader(data)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("reading TypeScript with node: %v\n%s", err, stderr.String())
	}
	var o outline
	if err := json.Unmarshal(out, &o); err != nil {
		return nil, fmt.Errorf("reading the TypeScript outline: %v", err)
	}
	return &o, nil
}

type reader struct {
	reverse.Reader
	path  string
	stmts []*statement

	// names maps each TypeScript type import reads to its TDL name.
	names map[string]string
	// sums is the unions read as fielded enums, and variants the
	// interfaces they absorb.
	sums     map[string]*sum
	variants map[string]bool
}

// sum is a union read as a fielded enum: its discriminant, and each
// member interface with its tag. A carried union has no discriminant.
type sum struct {
	disc    string
	members []string
	tags    []string
}

func (r *reader) read(ctx context.Context, files []*plugin.File) (*ir.Model, error) {
	if len(files) != 1 {
		return nil, reverse.Failf(nil, "import reads one .ts file, and was given %d", len(files))
	}
	r.path = files[0].GetPath()
	if !strings.HasSuffix(r.path, ".ts") {
		return nil, reverse.Failf(r.pos(0, 0), "import reads .ts files, and %s is not one", r.path)
	}
	o, err := parse(ctx, files)
	if errors.Is(err, ErrNoCompiler) {
		return nil, reverse.Failf(nil, "%v", err)
	}
	if err != nil {
		return nil, reverse.Failf(r.pos(0, 0), "%v", err)
	}
	if len(o.Files) != 1 {
		return nil, reverse.Failf(r.pos(0, 0), "the TypeScript outline holds %d files", len(o.Files))
	}
	f := o.Files[0]
	if len(f.Diagnostics) > 0 {
		d := f.Diagnostics[0]
		return nil, reverse.Failf(r.pos(d.Line, d.Col), "%s does not parse: %s", r.path, d.Message)
	}
	r.stmts = f.Statements

	docs := map[*statement]*jsdoc{}
	for _, s := range r.stmts {
		doc, err := r.jsdoc(s.Docs, s.Line, s.Col)
		if err != nil {
			return nil, err
		}
		docs[s] = doc
		r.Roundtrip = r.Roundtrip || len(doc.tdl) > 0
	}

	file := &ast.File{Filename: r.path}
	pkg, items, pdoc, err := r.header(file, docs)
	if err != nil {
		return nil, err
	}
	if pkg != "" || len(pdoc) > 0 {
		file.Package = &ast.PackageDecl{Path: pkg, Doc: pdoc, DocP: make([]ast.Position, len(pdoc))}
	}

	carried := map[*statement]ast.Decl{}
	for _, s := range r.stmts {
		src, ok := get(docs[s].tdl, "source")
		if !ok {
			continue
		}
		decl, err := reverse.ParseItem(src)
		if err != nil {
			return nil, reverse.Failf(r.pos(s.Line, s.Col), "the source of %s is not TDL: %v", s.Name, err)
		}
		carried[s] = decl
	}
	r.nameAll(carried)

	var emitted []reverse.Item
	for _, s := range r.stmts {
		decl, err := r.statement(s, docs[s], carried[s])
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

func (r *reader) pos(line, col int) *ir.Position {
	return &ir.Position{Filename: r.path, Line: int32(line), Column: int32(col)}
}

func (r *reader) at(line, col int) ast.Position {
	return ast.Position{Filename: r.path, Line: line, Col: col}
}

// packageBack is the package a file's name reads back as: its base name,
// unless it is the one a model without a package is written to.
func packageBack(file string) string {
	name := strings.TrimSuffix(strings.TrimSuffix(path.Base(file), ".ts"), ".d")
	if name == defaultFile {
		return ""
	}
	return name
}

// header reads what the file says of the whole model: the package, from
// the file's name, and under roundtrip the file annotations.
func (r *reader) header(file *ast.File, docs map[*statement]*jsdoc) (string, []reverse.Item, []string, error) {
	pkg := packageBack(r.path)
	if pkg != "" && !tdlName(pkg) {
		r.Warn(emit.LossName, r.pos(0, 0), "%s names no package, and regenerates as %s.ts", r.path, defaultFile)
		pkg = ""
	}

	var anns []kv
	for _, s := range r.stmts {
		anns = append(anns, docs[s].tdl...)
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
			return "", nil, nil, reverse.Failf(r.pos(0, 0), "@tdl import %q is not one TDL import: %v", src, err)
		}
		file.Imports = append(file.Imports, imp)
	}
	var items []reverse.Item
	for _, v := range all(anns, "item") {
		var at int
		if n, err := fmt.Sscanf(v, "%d", &at); n != 1 || err != nil {
			return "", nil, nil, reverse.Failf(r.pos(0, 0), "a @tdl item does not start with its index")
		}
		_, src, _ := strings.Cut(v, " ")
		decl, err := reverse.ParseItem(src)
		if err != nil {
			return "", nil, nil, reverse.Failf(r.pos(0, 0), "a @tdl item is not TDL: %v", err)
		}
		items = append(items, reverse.Item{Decl: decl, At: at})
	}
	return pkg, items, doc, nil
}

// declares reports whether a statement declares a type import reads.
func declares(s *statement) bool {
	return s.Name != "" && (s.Kind == "interface" || s.Kind == "type" || s.Kind == "enum")
}

// nameAll gives every type import reads its TDL name before any is read,
// since a property may name one declared after it, and finds the unions
// written as fielded enums.
func (r *reader) nameAll(carried map[*statement]ast.Decl) {
	ifaces := map[string]*statement{}
	for _, s := range r.stmts {
		if !declares(s) {
			continue
		}
		r.names[s.Name] = s.Name
		if d := carried[s]; d != nil {
			r.names[s.Name] = d.Name()
		}
		if s.Kind == "interface" {
			ifaces[s.Name] = s
		}
	}

	uses := map[string]int{}
	var count func(t *tsType)
	count = func(t *tsType) {
		if t == nil {
			return
		}
		if t.K == "ref" {
			uses[t.Name]++
		}
		for _, x := range slices.Concat(t.Args, t.Types, []*tsType{t.Elem}) {
			count(x)
		}
	}
	for _, s := range r.stmts {
		count(s.Type)
		for _, m := range s.Members {
			count(m.Type)
		}
	}

	for _, s := range r.stmts {
		if s.Kind != "type" || s.Name == "" || s.Params > 0 {
			continue
		}
		var members []string
		for _, t := range unionOf(s.Type) {
			if t.K != "ref" || len(t.Args) > 0 {
				members = nil
				break
			}
			members = append(members, t.Name)
		}
		if len(members) == 0 {
			continue
		}
		var sm *sum
		if _, ok := carried[s].(*ast.EnumDecl); ok {
			sm = &sum{members: members}
		} else if carried[s] == nil {
			sm = r.tagged(members, ifaces, uses)
		}
		if sm == nil || slices.ContainsFunc(members, func(m string) bool { return r.variants[m] }) {
			continue
		}
		r.sums[s.Name] = sm
		for _, m := range members {
			r.variants[m] = true
			delete(r.names, m)
		}
	}
}

// unionOf is a union's members, or the type alone.
func unionOf(t *tsType) []*tsType {
	if t == nil {
		return nil
	}
	if t.K == "union" {
		return t.Types
	}
	return []*tsType{t}
}

// tagged finds the property every member interface tags itself with, a
// string literal naming its variant, or returns nil when the members are
// not a fielded enum's.
func (r *reader) tagged(members []string, ifaces map[string]*statement, uses map[string]int) *sum {
	for _, m := range members {
		s := ifaces[m]
		if s == nil || uses[m] != 1 || s.Params > 0 || s.Heritage || !s.Exported {
			return nil
		}
	}
	for _, cand := range ifaces[members[0]].Members {
		if cand.Kind != "property" || cand.Optional || cand.Type.K != "lit" || cand.Type.Str == nil {
			continue
		}
		sm := &sum{disc: cand.Name, members: members}
		seen := map[string]bool{}
		for _, m := range members {
			tag := ""
			for _, p := range ifaces[m].Members {
				if p.Kind == "property" && p.Name == cand.Name && !p.Optional && p.Type.K == "lit" && p.Type.Str != nil {
					tag = *p.Type.Str
				}
			}
			if !tdlName(tag) || seen[tag] {
				sm = nil
				break
			}
			seen[tag] = true
			sm.tags = append(sm.tags, tag)
		}
		if sm != nil {
			return sm
		}
	}
	return nil
}

var tdlIdent = regexp.MustCompile(`^` + lex.IdentPattern + `$`)

// tdlName reports whether s can name a TDL declaration or variant.
func tdlName(s string) bool {
	if !tdlIdent.MatchString(s) {
		return false
	}
	_, keyword := lex.Lookup(s)
	return !keyword
}

// head is a node's head from its JSDoc.
func (r *reader) head(name string, doc *jsdoc, line, col int) ast.DeclHead {
	h := reverse.Head(name, doc.doc, r.at(line, col), false)
	h.Dep = doc.deprecated
	return h
}

// jsdoc reads a node's JSDoc comments, the last of which is its doc
// comment, and warns about what import does not read.
func (r *reader) jsdoc(docs []string, line, col int) (*jsdoc, error) {
	if len(docs) == 0 {
		return &jsdoc{}, nil
	}
	if len(docs) > 1 {
		r.Warn(emit.LossDoc, r.pos(line, col), "only the last of %d JSDoc comments is read", len(docs))
	}
	doc, err := parseJSDoc(docs[len(docs)-1])
	if err != nil {
		return nil, reverse.Failf(r.pos(line, col), "%v", err)
	}
	for _, tag := range doc.other {
		r.Warn(emit.LossDoc, r.pos(line, col), "the JSDoc tag @%s is not read", tag)
	}
	return doc, nil
}

// statement reads one top-level statement, or returns nil for one with no
// declaration of its own.
func (r *reader) statement(s *statement, doc *jsdoc, carried ast.Decl) (ast.Decl, error) {
	if carried != nil {
		return carried, nil
	}
	if s.Kind == "empty-export" {
		return nil, nil
	}
	if !declares(s) {
		r.Warn(emit.LossUnsupported, r.pos(s.Line, s.Col), "%s has no TDL form", syntax(s.Syntax))
		return nil, nil
	}
	if r.variants[s.Name] {
		return nil, nil
	}
	pos := r.pos(s.Line, s.Col)
	if s.Params > 0 {
		r.Warn(emit.LossGeneric, pos, "%s takes type parameters, which are not read", s.Name)
		return nil, nil
	}
	if !s.Exported || s.Declare {
		r.Warn(emit.LossUnsupported, pos, "%s regenerates as an exported declaration", s.Name)
	}
	name := r.names[s.Name]
	if !tdlName(name) {
		r.Warn(emit.LossUnsupported, pos, "%s is not a TDL name", s.Name)
		return nil, nil
	}
	if emit.Pascal(name) != name {
		r.Directive(name, "name", reverse.StrLit(s.Name))
	}
	head := r.head(name, doc, s.Line, s.Col)

	switch s.Kind {
	case "interface":
		if s.Heritage {
			r.Warn(emit.LossUnsupported, pos, "%s extends another type, which is not read", s.Name)
		}
		d := &ast.StructDecl{DeclHead: head, Keyword: "type"}
		fields, err := r.fields(s.Members, name, "")
		if err != nil {
			return nil, err
		}
		for _, f := range fields {
			d.Members = append(d.Members, f)
		}
		if len(d.Members) > 0 {
			d.Rbrace = d.P
		}
		return d, nil
	case "enum":
		r.Warn(emit.LossUnsupported, pos, "enum %s regenerates as a union of string literals", s.Name)
		d := &ast.EnumDecl{DeclHead: head}
		for _, m := range s.Members {
			if !tdlName(m.Name) {
				r.Warn(emit.LossUnsupported, r.pos(m.Line, m.Col), "%s.%s is not a TDL name", s.Name, m.Name)
				continue
			}
			vdoc, err := r.jsdoc(m.Docs, m.Line, m.Col)
			if err != nil {
				return nil, err
			}
			d.Variants = append(d.Variants, &ast.Variant{DeclHead: r.head(m.Name, vdoc, m.Line, m.Col)})
		}
		return d, nil
	}

	if sm := r.sums[s.Name]; sm != nil {
		return r.union(s, head, sm)
	}
	return r.alias(s, head)
}

// syntax names a statement's kind in a message: "FunctionDeclaration" is
// "a function declaration".
func syntax(kind string) string {
	words := emit.Words(kind)
	for i, w := range words {
		words[i] = strings.ToLower(w)
	}
	text := strings.Join(words, " ")
	if strings.ContainsRune("aeiou", rune(text[0])) {
		return "an " + text
	}
	return "a " + text
}

// alias reads a type alias: a union of string literals naming variants is
// an enum, `never` an enum with none, and anything else a newtype.
func (r *reader) alias(s *statement, head ast.DeclHead) (ast.Decl, error) {
	if s.Type.K == "never" {
		return &ast.EnumDecl{DeclHead: head}, nil
	}
	members := unionOf(s.Type)
	enum := &ast.EnumDecl{DeclHead: head}
	for _, t := range members {
		if t.K != "lit" || t.Str == nil || !tdlName(*t.Str) || slices.ContainsFunc(enum.Variants, func(v *ast.Variant) bool { return v.N == *t.Str }) {
			enum = nil
			break
		}
		enum.Variants = append(enum.Variants, &ast.Variant{DeclHead: ast.DeclHead{N: *t.Str, P: head.P}})
	}
	if enum != nil {
		return enum, nil
	}

	typ, cs := r.typeOf(s.Type, s.Name, s.Line, s.Col, true)
	if typ == nil {
		return nil, nil
	}
	d := &ast.NewtypeDecl{DeclHead: head, Base: typ, Constraints: cs}
	if len(cs) > 0 {
		d.Rbrace = d.P
	}
	return d, nil
}

// union reads a union of interfaces as a fielded enum, each interface a
// variant named by its tag.
func (r *reader) union(s *statement, head ast.DeclHead, sm *sum) (ast.Decl, error) {
	name := head.N
	d := &ast.EnumDecl{DeclHead: head}
	if sm.disc != emit.DefaultDiscriminant {
		r.Directive(name, "discriminant", reverse.StrLit(sm.disc))
	}
	for i, m := range sm.members {
		iface := r.find(m)
		vname := sm.tags[i]
		path := name + "." + vname
		if m != s.Name+emit.Pascal(vname) {
			r.Directive(path, "name", reverse.StrLit(m))
		}
		doc, err := r.jsdoc(iface.Docs, iface.Line, iface.Col)
		if err != nil {
			return nil, err
		}
		v := &ast.Variant{DeclHead: r.head(vname, doc, iface.Line, iface.Col)}
		v.Fields, err = r.fields(iface.Members, path, sm.disc)
		if err != nil {
			return nil, err
		}
		if len(v.Fields) > 0 {
			v.Rbrace = v.P
		}
		d.Variants = append(d.Variants, v)
	}
	return d, nil
}

func (r *reader) find(name string) *statement {
	for _, s := range r.stmts {
		if s.Name == name && s.Kind == "interface" {
			return s
		}
	}
	return nil
}

// fields reads an interface's properties, skipping the discriminant and
// any member with no TDL form.
func (r *reader) fields(members []*member, owner, disc string) ([]*ast.Field, error) {
	var out []*ast.Field
	for _, m := range members {
		if m.Kind != "property" {
			r.Warn(emit.LossUnsupported, r.pos(m.Line, m.Col), "%s's %s has no TDL form", owner, strings.TrimPrefix(syntax(m.Syntax), "a "))
			continue
		}
		if disc != "" && m.Name == disc {
			continue
		}
		path := owner + "." + m.Name
		name := m.Name
		if !tdlIdent.MatchString(name) {
			name = emit.Camel(name)
			if !tdlIdent.MatchString(name) {
				r.Warn(emit.LossUnsupported, r.pos(m.Line, m.Col), "%s has no TDL name", path)
				continue
			}
			path = owner + "." + name
			r.Directive(path, "name", reverse.StrLit(m.Name))
		}
		if m.Readonly {
			r.Warn(emit.LossUnsupported, r.pos(m.Line, m.Col), "%s is readonly, which is not read", path)
		}
		typ, cs := r.typeOf(m.Type, path, m.Line, m.Col, true)
		if typ == nil {
			continue
		}
		typ.Optional = m.Optional
		doc, err := r.jsdoc(m.Docs, m.Line, m.Col)
		if err != nil {
			return nil, err
		}
		f := &ast.Field{DeclHead: r.head(name, doc, m.Line, m.Col), Type: typ, Constraints: cs}
		if len(cs) > 0 {
			f.Rbrace = f.P
		}
		out = append(out, f)
	}
	return out, nil
}

// typeOf reads a type, or warns and returns nil. top is whether the type
// is a property's or an alias's, where `| null` is a nullable value and a
// union of literals a constraint; within a collection, `| null` is an
// optional element.
func (r *reader) typeOf(t *tsType, what string, line, col int, top bool) (*ast.TypeRef, []*ast.Constraint) {
	var members, nulls []*tsType
	for _, m := range unionOf(t) {
		if m.K == "null" {
			nulls = append(nulls, m)
		} else {
			members = append(members, m)
		}
	}
	var out *ast.TypeRef
	var cs []*ast.Constraint
	switch {
	case len(members) == 1 && members[0].K != "lit":
		out = r.single(members[0], what, line, col)
	case top && len(members) > 0:
		out, cs = r.literals(members)
	}
	if out == nil {
		if len(members) > 0 && (len(members) > 1 || members[0].K == "lit") {
			r.Warn(emit.LossUnsupported, r.pos(line, col), "%s is a union TDL has no form for", what)
		}
		return nil, nil
	}
	if len(nulls) > 0 {
		if top {
			out.Nullable = true
		} else {
			out.Optional = true
		}
	}
	return out, cs
}

// literals reads a union of string or of number literals as a string or
// a float64 constrained to them, or returns nil.
func (r *reader) literals(members []*tsType) (*ast.TypeRef, []*ast.Constraint) {
	oneOf := &ast.Constraint{N: "oneOf"}
	str := members[0].Str != nil
	for _, m := range members {
		switch {
		case m.K != "lit":
			return nil, nil
		case str && m.Str != nil:
			oneOf.Args = append(oneOf.Args, &ast.Literal{Kind: ast.LitString, Text: *m.Str})
		case !str && m.Num != nil:
			kind := ast.LitInt
			if strings.ContainsAny(*m.Num, ".eE") {
				kind = ast.LitFloat
			}
			if _, err := strconv.ParseFloat(*m.Num, 64); err != nil {
				return nil, nil
			}
			oneOf.Args = append(oneOf.Args, &ast.Literal{Kind: kind, Text: *m.Num})
		default:
			return nil, nil
		}
	}
	if str {
		return &ast.TypeRef{N: "string"}, []*ast.Constraint{oneOf}
	}
	return &ast.TypeRef{N: "float64"}, []*ast.Constraint{oneOf}
}

// single reads a type that is not a union.
func (r *reader) single(t *tsType, what string, line, col int) *ast.TypeRef {
	switch t.K {
	case "string":
		return &ast.TypeRef{N: "string"}
	case "number":
		return &ast.TypeRef{N: "float64"}
	case "boolean":
		return &ast.TypeRef{N: "bool"}
	case "array":
		elem, _ := r.typeOf(t.Elem, what, line, col, false)
		if elem == nil {
			return nil
		}
		return &ast.TypeRef{List: elem}
	case "ref":
		return r.ref(t, what, line, col)
	}
	text := t.Text
	if text == "" {
		text = t.K
	}
	r.Warn(emit.LossUnsupported, r.pos(line, col), "%s is %s, which TDL has no form for", what, text)
	return nil
}

// ref reads a type reference: Array, Record, Partial<Record>, or a type
// the file declares.
func (r *reader) ref(t *tsType, what string, line, col int) *ast.TypeRef {
	switch {
	case (t.Name == "Array" || t.Name == "ReadonlyArray") && len(t.Args) == 1:
		elem, _ := r.typeOf(t.Args[0], what, line, col, false)
		if elem == nil {
			return nil
		}
		return &ast.TypeRef{List: elem}
	case t.Name == "Partial" && len(t.Args) == 1 && t.Args[0].K == "ref" && t.Args[0].Name == "Record":
		return r.ref(t.Args[0], what, line, col)
	case t.Name == "Record" && len(t.Args) == 2:
		key, _ := r.typeOf(t.Args[0], what, line, col, false)
		val, _ := r.typeOf(t.Args[1], what, line, col, false)
		if key == nil || val == nil {
			return nil
		}
		return &ast.TypeRef{MapKey: key, MapValue: val}
	case len(t.Args) == 0 && r.names[t.Name] != "":
		return &ast.TypeRef{N: r.names[t.Name]}
	}
	r.Warn(emit.LossUnsupported, r.pos(line, col), "%s names %s, which is not imported", what, t.Name)
	return nil
}

// kv is one @tdl tag.
type kv struct{ key, value string }

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

// jsdoc is a JSDoc comment as import reads it.
type jsdoc struct {
	doc        []string
	deprecated *ast.Deprecation
	tdl        []kv
	// other is the name of each tag import does not read.
	other []string
}

// parseJSDoc reads a /** */ comment: the lines before its first tag are
// the doc comment, an @deprecated tag the deprecation, its text the
// reason, and each @tdl tag an annotation, a key and a Go string literal.
func parseJSDoc(raw string) (*jsdoc, error) {
	body := strings.TrimSuffix(strings.TrimPrefix(raw, "/**"), "*/")
	var lines []string
	for i, line := range strings.Split(body, "\n") {
		if i > 0 {
			line = strings.TrimLeft(line, " \t")
			if rest, ok := strings.CutPrefix(line, "*"); ok {
				line = rest
			}
		}
		line = strings.TrimPrefix(line, " ")
		lines = append(lines, strings.TrimRight(line, " \t\r"))
	}
	lines = trimBlank(lines)

	out := &jsdoc{}
	var tags [][]string
	for _, line := range lines {
		switch {
		case strings.HasPrefix(line, "@"):
			tags = append(tags, []string{line})
		case len(tags) > 0:
			tags[len(tags)-1] = append(tags[len(tags)-1], line)
		default:
			out.doc = append(out.doc, line)
		}
	}
	out.doc = trimBlank(out.doc)

	for _, tag := range tags {
		name, first, _ := strings.Cut(strings.TrimPrefix(tag[0], "@"), " ")
		text := strings.Join(trimBlank(append([]string{first}, tag[1:]...)), "\n")
		switch name {
		case "deprecated":
			out.deprecated = &ast.Deprecation{Reason: text}
		case "tdl":
			key, quoted, _ := strings.Cut(text, " ")
			value, err := strconv.Unquote(quoted)
			if err != nil {
				return nil, fmt.Errorf("@tdl %s does not hold a quoted value", key)
			}
			out.tdl = append(out.tdl, kv{key, value})
		default:
			out.other = append(out.other, name)
		}
	}
	return out, nil
}

// trimBlank drops leading and trailing empty lines.
func trimBlank(lines []string) []string {
	for len(lines) > 0 && lines[0] == "" {
		lines = lines[1:]
	}
	for len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}

// arrays writes Array<X> and ReadonlyArray<X> as X[], which reads the
// same.
func arrays(t *tsType) *tsType {
	if t == nil {
		return nil
	}
	if t.K == "ref" && (t.Name == "Array" || t.Name == "ReadonlyArray") && len(t.Args) == 1 {
		return &tsType{K: "array", Elem: arrays(t.Args[0])}
	}
	for i := range t.Args {
		t.Args[i] = arrays(t.Args[i])
	}
	for i := range t.Types {
		t.Types[i] = arrays(t.Types[i])
	}
	t.Elem = arrays(t.Elem)
	return t
}

// Normalize puts .ts files into the normal form the round-trip corpus
// compares: the compiler's outline of each, without positions, with each
// JSDoc comment read into its doc comment and tags, and without whether a
// property's name is quoted or an array is written Array<X>.
func Normalize(files []*plugin.File) ([]*plugin.File, error) {
	o, err := parse(context.Background(), files)
	if err != nil {
		return nil, err
	}
	normal := func(docs []string) []string {
		var out []string
		for _, d := range docs {
			doc, err := parseJSDoc(d)
			if err != nil {
				out = append(out, d)
				continue
			}
			data, _ := json.Marshal(struct {
				Doc        []string
				Deprecated *ast.Deprecation
				Tags       []string
			}{doc.doc, doc.deprecated, doc.other})
			out = append(out, string(data))
		}
		return out
	}
	out := make([]*plugin.File, len(o.Files))
	for i, f := range o.Files {
		for _, s := range f.Statements {
			s.Line, s.Col, s.Docs, s.Type = 0, 0, normal(s.Docs), arrays(s.Type)
			for _, m := range s.Members {
				m.Line, m.Col, m.Docs, m.Quoted, m.Type = 0, 0, normal(m.Docs), false, arrays(m.Type)
			}
		}
		data, err := json.MarshalIndent(f.Statements, "", "  ")
		if err != nil {
			return nil, err
		}
		out[i] = &plugin.File{Path: f.Path, Content: append(data, '\n')}
	}
	return out, nil
}
