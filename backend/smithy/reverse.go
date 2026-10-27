package smithy

import (
	"bytes"
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/unstoppablemango/tdl/ast"
	"github.com/unstoppablemango/tdl/backend/internal/emit"
	"github.com/unstoppablemango/tdl/backend/internal/reverse"
	"github.com/unstoppablemango/tdl/ir"
	"github.com/unstoppablemango/tdl/lex"
	"github.com/unstoppablemango/tdl/plugin"
)

// primitives maps a prelude shape to the TDL primitive it reads as: the
// inverse of preludeShapes, with Short and Byte widened.
var primitives = map[string]string{
	"String":     "string",
	"Integer":    "int32",
	"Long":       "int64",
	"BigInteger": "uint64",
	"Float":      "float32",
	"Double":     "float64",
	"Boolean":    "bool",
	"Blob":       "bytes",
	"BigDecimal": "decimal",
	"Timestamp":  "instant",
	"Short":      "int32",
	"Byte":       "int32",
}

// simpleShapes maps a simple shape's type to the prelude shape it holds.
var simpleShapes = map[string]string{
	"string":     "String",
	"integer":    "Integer",
	"long":       "Long",
	"bigInteger": "BigInteger",
	"float":      "Float",
	"double":     "Double",
	"boolean":    "Boolean",
	"blob":       "Blob",
	"bigDecimal": "BigDecimal",
	"timestamp":  "Timestamp",
	"short":      "Short",
	"byte":       "Byte",
}

// ErrNoCLI is why import cannot run: the Smithy CLI is not on PATH.
var ErrNoCLI = errors.New("importing Smithy runs the Smithy CLI, and needs smithy on PATH")

// CLI finds the Smithy CLI on PATH, or returns [ErrNoCLI].
func CLI() (string, error) {
	path, err := exec.LookPath("smithy")
	if err != nil {
		return "", ErrNoCLI
	}
	return path, nil
}

// Import reads .smithy files into a model: each structure a struct, each
// enum an enum, each union whose members target Unit or a structure
// nothing else names an enum with fields, and each simple shape, list, or
// map that is not a collection a member needs a newtype. The namespace is
// the package. What regenerating needs goes in a smithy target block.
// Under the tdl metadata and tdl#source traits a `roundtrip` directive
// writes, it rebuilds the model the files were generated from.
// docs/architecture/schema-backends.md has the mapping.
func (Backend) Import(ctx context.Context, req *plugin.ImportRequest) (*plugin.ImportResponse, error) {
	r := &reader{}
	return r.Response(r.read(ctx, req.GetFiles()))
}

// parse runs the Smithy CLI over files and returns the JSON AST it
// writes, with where each shape is declared.
func parse(ctx context.Context, files []*plugin.File) (*astModel, map[string]ast.Position, error) {
	cli, err := CLI()
	if err != nil {
		return nil, nil, err
	}
	dir, err := os.MkdirTemp("", "tdl-smithy-")
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = os.RemoveAll(dir) }()

	args := []string{"ast", "--no-config", "--quiet", "--allow-unknown-traits"}
	for _, f := range files {
		path := filepath.Join(dir, filepath.Clean(string(filepath.Separator)+f.GetPath()))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return nil, nil, err
		}
		if err := os.WriteFile(path, f.GetContent(), 0o600); err != nil {
			return nil, nil, err
		}
		args = append(args, path)
	}
	var stdout, stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, cli, args...)
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	cmd.Env = append(os.Environ(), "HOME="+dir)
	if err := cmd.Run(); err != nil {
		msg := strings.ReplaceAll(cliErrors(stderr.String()), dir+string(filepath.Separator), "")
		return nil, nil, reverse.Failf(nil, "the Smithy CLI rejects the files: %s", cmp.Or(msg, err.Error()))
	}
	m, err := parseAST(stdout.Bytes())
	if err != nil {
		return nil, nil, fmt.Errorf("reading the Smithy CLI's JSON AST: %w", err)
	}
	return m, locate(files), nil
}

// cliErrors is the Smithy CLI's standard error without the JVM's notices.
func cliErrors(stderr string) string {
	var lines []string
	for line := range strings.SplitSeq(stderr, "\n") {
		if strings.HasPrefix(line, "Picked up ") {
			continue
		}
		lines = append(lines, line)
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

var (
	namespaceStmt = regexp.MustCompile(`(?m)^[ \t]*namespace[ \t]+([A-Za-z0-9_.]+)`)
	shapeStmt     = regexp.MustCompile(`(?m)^[ \t]*(structure|union|enum|intEnum|list|set|map|string|blob|boolean|byte|short|integer|long|float|double|bigInteger|bigDecimal|timestamp|document|service|operation|resource)[ \t]+(` + `_*[A-Za-z][A-Za-z0-9_]*` + `)`)
	metadataStmt  = regexp.MustCompile(`(?m)^[ \t]*metadata[ \t]`)
)

// locate finds where each shape is declared, since the JSON AST has no
// positions: the line of its statement, keyed by its absolute ID.
func locate(files []*plugin.File) map[string]ast.Position {
	out := map[string]ast.Position{}
	for _, f := range files {
		src := f.GetContent()
		ns := namespaceStmt.FindSubmatch(src)
		if ns == nil {
			continue
		}
		for _, m := range shapeStmt.FindAllSubmatchIndex(src, -1) {
			id := string(ns[1]) + "#" + string(src[m[4]:m[5]])
			if _, ok := out[id]; ok {
				continue
			}
			line := bytes.Count(src[:m[4]], []byte("\n")) + 1
			col := m[2] - (bytes.LastIndexByte(src[:m[2]], '\n') + 1) + 1
			out[id] = ast.Position{Filename: f.GetPath(), Line: line, Col: col}
		}
	}
	return out
}

type reader struct {
	reverse.Reader

	// shapes is the model's own shapes, in the order they are declared.
	shapes []*shape
	byID   map[string]*shape
	at     map[string]ast.Position
	ns     string

	// names maps each shape read as a declaration to its TDL name.
	names map[string]string
	// sources is each shape's tdl#source, parsed.
	sources map[string]ast.Decl
	// variants is the structures union members absorb, and sugar the
	// lists and maps read as the collection they hold.
	variants map[string]bool
	sugar    map[string]bool
}

func (r *reader) read(ctx context.Context, files []*plugin.File) (*ir.Model, error) {
	if len(files) == 0 {
		return nil, reverse.Failf(nil, "import reads .smithy files, and was given none")
	}
	for _, f := range files {
		if filepath.Ext(f.GetPath()) != ".smithy" {
			return nil, reverse.Failf(nil, "import reads .smithy files, and was given %s", f.GetPath())
		}
	}
	m, at, err := parse(ctx, files)
	if errors.Is(err, ErrNoCLI) {
		return nil, reverse.Failf(nil, "%v", err)
	}
	if err != nil {
		return nil, err
	}

	meta, err := readMetadata(m.metadata)
	if err != nil {
		return nil, reverse.Failf(nil, "%v", err)
	}
	r.Roundtrip = meta != nil

	var namespaces []string
	var own []*shape
	for _, s := range m.shapes {
		if s.id == traitSource && s.traits.trait {
			continue
		}
		if !slices.Contains(namespaces, s.namespace()) {
			namespaces = append(namespaces, s.namespace())
		}
		own = append(own, s)
	}
	if len(namespaces) > 1 {
		return nil, reverse.Failf(nil, "import reads one namespace, and the files declare %s", strings.Join(namespaces, ", "))
	}
	ns := ""
	if len(namespaces) == 1 {
		ns = namespaces[0]
	}
	if err := r.prepare(own, at, ns); err != nil {
		return nil, err
	}

	file := &ast.File{Filename: files[0].GetPath()}
	pkg := ns
	var items []reverse.Item
	if meta != nil {
		if meta.Package != nil {
			pkg = *meta.Package
		}
		if meta.Doc != nil {
			doc := strings.Split(*meta.Doc, "\n")
			file.Package = &ast.PackageDecl{Path: pkg, Doc: doc, DocP: make([]ast.Position, len(doc))}
		}
		for _, src := range meta.Import {
			imp, err := reverse.ParseImport(src)
			if err != nil {
				return nil, reverse.Failf(nil, "the tdl metadata's import %q is not one TDL import: %v", src, err)
			}
			file.Imports = append(file.Imports, imp)
		}
		for _, v := range meta.Item {
			var n int
			if c, err := fmt.Sscanf(v, "%d", &n); c != 1 || err != nil {
				return nil, reverse.Failf(nil, "a tdl metadata item does not start with its index")
			}
			_, src, _ := strings.Cut(v, " ")
			decl, err := reverse.ParseItem(src)
			if err != nil {
				return nil, reverse.Failf(nil, "a tdl metadata item is not TDL: %v", err)
			}
			items = append(items, reverse.Item{Decl: decl, At: n})
		}
	}
	if pkg != "" && file.Package == nil {
		file.Package = &ast.PackageDecl{Path: pkg}
	}
	if !r.Roundtrip {
		for _, key := range slices.Sorted(maps.Keys(m.metadata)) {
			r.Warn(emit.LossUnsupported, nil, "metadata %s has no TDL form", key)
		}
	}

	var emitted []reverse.Item
	for _, s := range r.shapes {
		if decl := r.shape(s); decl != nil {
			emitted = append(emitted, reverse.Item{Decl: decl, At: -1})
		}
	}
	file.Decls = reverse.Arrange(emitted, items)
	if len(r.Entries) > 0 {
		file.Decls = append(file.Decls, &ast.TargetDecl{DeclHead: ast.DeclHead{N: Name}, For: pkg, Entries: r.Entries})
	}
	return reverse.Lower(file)
}

// metadataKey is the metadata a `roundtrip` directive writes.
const metadataKey = "tdl"

// annotations is the tdl metadata: what no shape carries.
type annotations struct {
	Package *string  `json:"package,omitempty"`
	Doc     *string  `json:"doc,omitempty"`
	Import  []string `json:"import,omitempty"`
	Item    []string `json:"item,omitempty"`
}

// readMetadata reads the tdl metadata, or returns nil when there is none.
func readMetadata(metadata map[string]json.RawMessage) (*annotations, error) {
	raw, ok := metadata[metadataKey]
	if !ok {
		return nil, nil
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	var a annotations
	if err := dec.Decode(&a); err != nil {
		return nil, fmt.Errorf("the tdl metadata is not what roundtrip writes: %v", err)
	}
	return &a, nil
}

// prepare orders the shapes by where they are declared and decides how
// each reads: its TDL name, and whether it is a variant's payload or a
// collection a member holds.
func (r *reader) prepare(shapes []*shape, at map[string]ast.Position, ns string) error {
	r.ns, r.at = ns, at
	r.byID = map[string]*shape{}
	r.names = map[string]string{}
	r.sources = map[string]ast.Decl{}
	r.variants = map[string]bool{}
	r.sugar = map[string]bool{}

	r.shapes = slices.Clone(shapes)
	index := map[string]int{}
	for i, s := range shapes {
		index[s.id] = i
	}
	slices.SortStableFunc(r.shapes, func(a, b *shape) int {
		pa, oka := at[a.id]
		pb, okb := at[b.id]
		switch {
		case oka && okb:
			return cmp.Or(cmp.Compare(pa.Filename, pb.Filename), cmp.Compare(pa.Line, pb.Line), cmp.Compare(pa.Col, pb.Col))
		case oka != okb:
			if oka {
				return -1
			}
			return 1
		}
		return cmp.Compare(index[a.id], index[b.id])
	})

	uses := map[string]int{}
	for _, s := range r.shapes {
		r.byID[s.id] = s
		for _, t := range s.targets() {
			uses[t]++
		}
	}
	for _, s := range r.shapes {
		if s.traits.source == nil {
			r.names[s.id] = s.name()
			continue
		}
		decl, err := reverse.ParseItem(*s.traits.source)
		if err != nil {
			return reverse.Failf(r.pos(s.id), "the tdl#source of %s is not TDL: %v", s.name(), err)
		}
		if !emit.IsDecl(decl) {
			return reverse.Failf(r.pos(s.id), "the tdl#source of %s is not a declaration", s.name())
		}
		r.sources[s.id] = decl
		r.names[s.id] = decl.Name()
	}

	// A union reads as an enum only when every member does, and only then
	// are the structures its members target its variants'.
	for _, s := range r.shapes {
		if s.kind != "union" {
			continue
		}
		var own []string
		for _, m := range s.members {
			if m.target == prelude+"#Unit" {
				continue
			}
			t := r.byID[m.target]
			if t == nil || t.kind != "structure" || uses[t.id] != 1 || !r.payload(t) {
				own = nil
				if r.sources[s.id] == nil {
					delete(r.names, s.id)
				}
				break
			}
			own = append(own, t.id)
		}
		for _, id := range own {
			r.variants[id] = true
			delete(r.names, id)
		}
	}
	for _, s := range r.shapes {
		if r.collection(s) {
			r.sugar[s.id] = true
			delete(r.names, s.id)
		}
	}

	// A shape with no TDL form declares nothing, so a name is kept only
	// for one that reads, and a collection holding what does not read
	// declares nothing in turn.
	for _, s := range r.shapes {
		if _, simple := simpleShapes[s.kind]; r.sources[s.id] == nil && !simple && !slices.Contains(declaring, s.kind) {
			delete(r.names, s.id)
		}
	}
	for changed := true; changed; {
		changed = false
		for _, s := range r.shapes {
			if r.names[s.id] == "" || r.sources[s.id] != nil || s.kind != "list" && s.kind != "map" {
				continue
			}
			if !r.reads(s, map[string]bool{}) {
				delete(r.names, s.id)
				changed = true
			}
		}
	}
	return nil
}

// declaring is the kinds of shape besides simple shapes that read as a
// declaration.
var declaring = []string{"structure", "union", "enum", "intEnum", "list", "map"}

// reads reports whether everything a list or map holds reads as a type.
func (r *reader) reads(s *shape, seen map[string]bool) bool {
	if seen[s.id] {
		return false
	}
	seen[s.id] = true
	for _, target := range s.targets() {
		ns, name, _ := strings.Cut(target, "#")
		t := r.byID[target]
		switch {
		case ns == prelude:
			if primitives[name] == "" {
				return false
			}
		case t == nil:
			return false
		case r.sugar[target]:
			if !r.reads(t, seen) {
				return false
			}
		case r.names[target] == "":
			return false
		}
	}
	return true
}

// payload reports whether a structure can be a variant's: one carrying
// nothing but its members and a doc comment.
func (r *reader) payload(s *shape) bool {
	t := s.traits
	return t.deprecated == nil && t.source == nil && len(t.other) == 0
}

// collection reports whether a list or map is one Generate writes for a
// member: named for what it holds, carrying no trait but @sparse and
// @uniqueItems.
func (r *reader) collection(s *shape) bool {
	t := s.traits
	if t.doc != nil || t.deprecated != nil || t.source != nil || len(t.other) > 0 || s.namespace() != r.ns {
		return false
	}
	switch s.kind {
	case "list":
		return s.member != nil && len(s.member.traits.other) == 0 && s.name() == listName(s.member.target, t.sparse, t.unique)
	case "map":
		return s.key != nil && s.value != nil && !t.unique && len(s.key.traits.other) == 0 && len(s.value.traits.other) == 0 &&
			s.name() == mapName(s.key.target, s.value.target, t.sparse)
	}
	return false
}

func (r *reader) pos(id string) *ir.Position {
	p, ok := r.at[id]
	if !ok {
		return nil
	}
	return &ir.Position{Filename: p.Filename, Line: int32(p.Line), Column: int32(p.Col)}
}

// shape reads one shape, or returns nil for one with no declaration of
// its own.
func (r *reader) shape(s *shape) ast.Decl {
	if r.variants[s.id] || r.sugar[s.id] {
		return nil
	}
	if decl, ok := r.sources[s.id]; ok {
		return decl
	}
	r.unknown(s.id, s.name(), s.traits)

	name := r.names[s.id]
	var decl ast.Decl
	switch s.kind {
	case "structure":
		fields := r.fields(s, name)
		st := &ast.StructDecl{DeclHead: r.head(name, s.id, s.traits), Keyword: "type", Members: make([]ast.Member, len(fields))}
		for i, f := range fields {
			st.Members[i] = f
		}
		if len(fields) > 0 {
			st.Rbrace = st.P
		}
		decl = st
	case "enum", "intEnum":
		decl = r.enum(s, name)
	case "union":
		u := r.union(s, name)
		if u == nil {
			return nil
		}
		decl = u
	case "list", "map":
		base := r.collectionRef(s, name)
		if base == nil {
			return nil
		}
		decl = &ast.NewtypeDecl{DeclHead: r.head(name, s.id, s.traits), Base: base}
	default:
		prim, ok := simpleShapes[s.kind]
		if !ok {
			r.Warn(emit.LossUnsupported, r.pos(s.id), "%s %s has no TDL form", s.kind, s.name())
			return nil
		}
		base := r.primitive(prim, s.id, name)
		decl = &ast.NewtypeDecl{DeclHead: r.head(name, s.id, s.traits), Base: base}
	}
	if emit.Pascal(s.name()) != s.name() {
		r.Directive(name, "name", reverse.StrLit(s.name()))
	}
	return decl
}

// unknown warns about each trait with no TDL form.
func (r *reader) unknown(id, what string, t traits) {
	other := slices.Clone(t.other)
	slices.Sort(other)
	for _, trait := range other {
		if trait == "mixins" {
			r.Warn(emit.LossUnsupported, r.pos(id), "%s uses mixins, which are not read", what)
			continue
		}
		r.Warn(emit.LossUnsupported, r.pos(id), "%s's trait @%s has no TDL form", what, trait)
	}
}

// head is a node's head: its doc comment and deprecation.
func (r *reader) head(name, id string, t traits) ast.DeclHead {
	var doc []string
	if t.doc != nil {
		doc = strings.Split(*t.doc, "\n")
	}
	h := reverse.Head(name, doc, r.at[id], false)
	if t.deprecated != nil {
		h.Dep = &ast.Deprecation{Reason: t.deprecated.message}
	}
	return h
}

// fields reads a structure's members, skipping one with no TDL form.
func (r *reader) fields(s *shape, owner string) []*ast.Field {
	var out []*ast.Field
	for _, m := range s.members {
		what := owner + "." + m.name
		r.unknown(s.id, what, m.traits)
		typ := r.typeRef(m.target, s.id, what)
		if typ == nil {
			continue
		}
		typ.Optional = !m.traits.required
		if !tdlIdent.MatchString(m.name) {
			r.Warn(emit.LossUnsupported, r.pos(s.id), "%s is not a TDL name, and is not read", what)
			continue
		}
		f := &ast.Field{DeclHead: r.head(m.name, s.id, m.traits), Type: typ}
		out = append(out, f)
	}
	return out
}

func (r *reader) enum(s *shape, name string) *ast.EnumDecl {
	decl := &ast.EnumDecl{DeclHead: r.head(name, s.id, s.traits)}
	if s.kind == "intEnum" {
		r.Warn(emit.LossUnsupported, r.pos(s.id), "intEnum %s's values are not read, and it regenerates as an enum", s.name())
	}
	for _, m := range s.members {
		r.unknown(s.id, name+"."+m.name, m.traits)
		vname := emit.FromScreaming(m.name)
		if v := m.traits.enumValue; v != nil {
			if tdlName(*v) {
				vname = *v
			} else {
				r.Warn(emit.LossName, r.pos(s.id), "%s.%s's value %q is not a TDL name, and regenerates as %q", name, m.name, *v, vname)
			}
		}
		if emit.ScreamingSnake(vname) != m.name {
			r.Directive(name+"."+vname, "name", reverse.StrLit(m.name))
		}
		decl.Variants = append(decl.Variants, &ast.Variant{DeclHead: r.head(vname, s.id, m.traits)})
	}
	return decl
}

// union reads a union whose members target Unit or a variant's structure,
// and skips any other.
func (r *reader) union(s *shape, name string) *ast.EnumDecl {
	for _, m := range s.members {
		if m.target != prelude+"#Unit" && !r.variants[m.target] {
			r.Warn(emit.LossUnsupported, r.pos(s.id), "union %s's member %s targets %s, which is not a structure only it names, and TDL has no form for it", s.name(), m.name, m.target)
			return nil
		}
	}
	decl := &ast.EnumDecl{DeclHead: r.head(name, s.id, s.traits)}
	for _, m := range s.members {
		vname := emit.Pascal(m.name)
		path := name + "." + vname
		r.unknown(s.id, path, m.traits)
		if emit.Camel(vname) != m.name {
			r.Warn(emit.LossName, r.pos(s.id), "union %s's member %s reads as variant %s, which regenerates as member %s", s.name(), m.name, vname, emit.Camel(vname))
		}
		vt := m.traits
		t := r.byID[m.target]
		if vt.doc == nil && t != nil {
			// Generate writes a variant's doc comment on both; a model
			// written by hand may put it on either.
			vt.doc = t.traits.doc
		}
		v := &ast.Variant{DeclHead: r.head(vname, s.id, vt)}
		if t != nil {
			r.unknown(t.id, t.name(), t.traits)
			if t.name() != s.name()+emit.Pascal(vname) {
				r.Directive(path, "name", reverse.StrLit(t.name()))
			}
			v.Fields = r.fields(t, path)
			if len(v.Fields) > 0 {
				v.Rbrace = v.P
			}
			if len(t.members) == 0 {
				r.Warn(emit.LossUnsupported, r.pos(t.id), "%s holds no members, and %s regenerates as fieldless", t.name(), path)
			}
		}
		decl.Variants = append(decl.Variants, v)
	}
	return decl
}

// typeRef reads the type a member targets, or warns and returns nil.
func (r *reader) typeRef(target, at, what string) *ast.TypeRef {
	ns, name, _ := strings.Cut(target, "#")
	if ns == prelude {
		return r.primitive(name, at, what)
	}
	s := r.byID[target]
	switch {
	case s == nil:
		r.Warn(emit.LossUnsupported, r.pos(at), "%s targets %s, which is not imported", what, target)
		return nil
	case r.sugar[target]:
		return r.collectionRef(s, what)
	case r.names[target] != "":
		return &ast.TypeRef{N: r.names[target]}
	}
	r.Warn(emit.LossUnsupported, r.pos(at), "%s targets %s, which has no TDL form", what, target)
	return nil
}

// primitive reads a prelude shape as a primitive, or warns and returns
// nil.
func (r *reader) primitive(name, at, what string) *ast.TypeRef {
	prim, ok := primitives[name]
	if !ok {
		r.Warn(emit.LossUnsupported, r.pos(at), "%s is a %s, which has no TDL form", what, name)
		return nil
	}
	if name == "Short" || name == "Byte" {
		r.Warn(emit.LossPrimitive, r.pos(at), "%s is a %s, which reads as int32 and regenerates as Integer", what, name)
	}
	return &ast.TypeRef{N: prim}
}

// collectionRef reads a list or map as the collection it holds, or warns
// and returns nil.
func (r *reader) collectionRef(s *shape, what string) *ast.TypeRef {
	if s.kind == "list" {
		elem := r.typeRef(s.member.target, s.id, what)
		if elem == nil {
			return nil
		}
		elem.Optional = s.traits.sparse
		if s.traits.unique {
			if s.traits.sparse {
				r.Warn(emit.LossUnsupported, r.pos(s.id), "%s is a sparse list of unique items, which TDL has no form for", s.name())
				return nil
			}
			return &ast.TypeRef{Set: elem}
		}
		return &ast.TypeRef{List: elem}
	}
	key := r.typeRef(s.key.target, s.id, what)
	value := r.typeRef(s.value.target, s.id, what)
	if key == nil || value == nil {
		return nil
	}
	value.Optional = s.traits.sparse
	return &ast.TypeRef{MapKey: key, MapValue: value}
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

// Normalize puts .smithy files into the normal form the round-trip corpus
// compares: the JSON AST of the shapes each declares, and of the metadata
// in the file that writes it.
func Normalize(files []*plugin.File) ([]*plugin.File, error) {
	m, at, err := parse(context.Background(), files)
	if err != nil {
		return nil, err
	}
	raw, err := rawShapes(m)
	if err != nil {
		return nil, err
	}
	out := make([]*plugin.File, len(files))
	for i, f := range files {
		var b bytes.Buffer
		if metadataStmt.Match(f.GetContent()) {
			meta, err := json.Marshal(m.metadata)
			if err != nil {
				return nil, err
			}
			fmt.Fprintf(&b, "metadata %s\n", meta)
		}
		for _, s := range m.shapes {
			if at[s.id].Filename == f.GetPath() {
				fmt.Fprintf(&b, "%s %s\n", s.id, raw[s.id])
			}
		}
		out[i] = &plugin.File{Path: f.GetPath(), Content: b.Bytes()}
	}
	return out, nil
}

// rawShapes is each shape's JSON AST, compacted, by ID.
func rawShapes(m *astModel) (map[string][]byte, error) {
	out := map[string][]byte{}
	for id, raw := range m.raw {
		var b bytes.Buffer
		if err := json.Compact(&b, raw); err != nil {
			return nil, err
		}
		out[id] = b.Bytes()
	}
	return out, nil
}
