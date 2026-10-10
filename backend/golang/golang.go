// Package golang generates Go source from a resolved model. A target block
// calls it "go". See docs/design/go-backend.md for the decisions.
//
// An enum whose variants carry no fields is a named string type with
// constants; one where any variant carries fields is a sealed interface with
// a struct per variant. decimal, uuid, and date map to placeholders. A class
// is an interface with one unexported marker method, which each satisfying
// declaration carries.
package golang

import (
	"context"
	"fmt"
	"go/format"
	"go/token"
	"maps"
	"slices"
	"strconv"
	"strings"

	"github.com/unstoppablemango/tdl/backend/internal/emit"
	"github.com/unstoppablemango/tdl/ir"
	"github.com/unstoppablemango/tdl/plugin"
)

// Name is the backend's name in a target block.
const Name = "go"

// Backend implements [plugin.Backend].
type Backend struct{}

func (Backend) Describe() plugin.Description {
	str := []ir.LiteralKind{ir.LiteralKind_LITERAL_KIND_STRING}
	return plugin.Description{
		Name:    Name,
		Version: "0.1.0",
		// Requests share no state.
		Reuse: true,
		Directives: []*plugin.DirectiveSpec{
			// The package clause, written as an import path.
			{Name: "package", MinArgs: 1, MaxArgs: 1, ArgKinds: str},
			{Name: "name", MinArgs: 1, MaxArgs: 1, ArgKinds: str},
			// A struct tag, emitted verbatim.
			{Name: "tag", MinArgs: 1, MaxArgs: 1, ArgKinds: str},
			// Field names identifying an entity. ArgKinds constrains by
			// position, so keyFields checks that each is a name.
			{Name: "key", MinArgs: 1, MaxArgs: -1},
			// An import path and the type that package declares.
			{Name: "foreign", MinArgs: 2, MaxArgs: 2, ArgKinds: []ir.LiteralKind{str[0], str[0]}},
			// One file for every declaration of the target, in the target block.
			{Name: "file", MinArgs: 1, MaxArgs: 1, ArgKinds: str},
			// Writes, as //tdl: comment directives, every fact the Go
			// source alone would lose, so import rebuilds the model.
			{Name: "roundtrip"},
		},
		Reverse: true,
	}
}

// generator holds the state for one request.
type generator struct {
	*emit.Session

	// cur is the index of the declaration being rendered.
	cur int32

	// needsComparable holds, per parameterized declaration, which of its
	// parameters Go needs to be comparable.
	needsComparable map[int32][]bool

	// classes is the shared class plan.
	classes *emit.InterfacePlan
	// genClass holds the classes generated as interfaces; marks holds the
	// classes whose marker each declaration carries. Both are by index.
	genClass map[int32]bool
	// classCycle holds the classes whose requires clause reaches themselves.
	classCycle map[int32]bool
	marks      map[int32][]int32

	// curClasses is the generated classes constraining each parameter of the
	// declaration being rendered.
	curClasses [][]int32

	// valid holds what has something to check and carries Validate.
	valid map[validKey]bool

	// imports maps each import path the current file uses to its alias, or
	// "" for none.
	imports map[string]string

	// foreign holds the declarations mapped to another package's type;
	// aliases holds the identifiers those imports are named by.
	foreign map[*ir.Decl]foreignType
	externs map[*ir.Extern]foreignType
	aliases map[string]bool

	// emitted is the declarations rendered, in model order, and reordered
	// whether import reads them in another.
	emitted   []*piece
	reordered bool
	// notes holds each declaration's annotations, and header the
	// package's.
	notes  map[*ir.Decl][]kv
	header []kv
}

// Generate returns one Go file per declaration the model owns, or one file
// holding them all when the target block names it.
func (Backend) Generate(_ context.Context, req *plugin.Request) (*plugin.Response, error) {
	g := &generator{Session: emit.NewSession(req, "Go")}
	g.Roundtrip = g.Bare("roundtrip")

	pkg := packageClause(g.Model.GetPackage())
	var pkgPos *ir.Position
	if d, ok := g.Block("package"); ok {
		pkg, pkgPos = packageClause(d.GetArgs()[0].GetText()), d.GetPosition()
	}

	// Every file carries the package clause, so a bad one is an error,
	// unless an annotation carries the package.
	if token.IsKeyword(pkg) {
		if !g.Roundtrip {
			g.Error(pkgPos, "%q is a Go keyword and cannot be a package name", pkg)
			return g.Response(nil), nil
		}
		pkg += "_"
	}
	g.packageLosses(pkg, pkgPos)

	// Rendering any declaration may consult these plans.
	g.planForeign()
	g.planClasses()
	g.inferComparable()
	g.planValidation()

	start := len(g.Diags)
	files := g.render(pkg)
	if g.Roundtrip {
		// What import would read from the files decides what the
		// annotations carry, so they are rendered again with them.
		g.simulate(files, pkg)
		g.Diags = g.Diags[:start]
		files = g.render(pkg)
	}
	return g.Response(files), nil
}

// render renders every declaration the model owns into files.
func (g *generator) render(pkg string) []*plugin.File {
	skipped := map[*ir.Decl]bool{}
	rendered := map[*ir.Decl]*piece{}
	// Each declaration's warnings are held until the cascade decides
	// whether it is emitted, since a skipped one loses everything at once.
	held := map[*ir.Decl][]*plugin.Diagnostic{}
	for i, decl := range g.Model.GetDecls() {
		if !emit.IsOwn(decl) {
			continue
		}
		mark := len(g.Diags)
		if d, ok := g.Find(decl.GetDirectives(), "file"); ok {
			g.Warn(emit.Unsupported(d.GetPosition(), "file names one file for the whole target, so it belongs in the target block"))
		}
		g.cur = int32(i)
		p, err := g.piece(decl)
		if err != nil {
			g.Diags = g.Diags[:mark]
			g.Skip(err)
			skipped[decl] = true
			continue
		}
		held[decl] = slices.Clone(g.Diags[mark:])
		g.Diags = g.Diags[:mark]
		if p != nil {
			rendered[decl] = p
		}
	}

	own := g.Own()
	mark := len(g.Diags)
	g.Cascade(own, skipped)
	if g.Roundtrip {
		// A skipped declaration is carried as TDL, so nothing is lost.
		g.Diags = g.Diags[:mark]
	}
	for _, d := range own {
		if !skipped[d] {
			g.Diags = append(g.Diags, held[d]...)
		}
	}

	var pieces []*piece
	for _, decl := range own {
		if p := rendered[decl]; p != nil && !skipped[decl] {
			pieces = append(pieces, p)
		}
	}
	g.emitted = pieces

	// One file holds the whole target when the block names it, and
	// otherwise each declaration has its own.
	type group struct {
		path   string
		pos    *ir.Position
		pieces []*piece
	}
	var groups []group
	if d, ok := g.Block("file"); ok {
		if len(pieces) > 0 || g.hasHeader() {
			groups = append(groups, group{d.GetArgs()[0].GetText(), d.GetPosition(), pieces})
		}
	} else {
		for _, p := range pieces {
			groups = append(groups, group{fileName(p.decl.GetMeta().GetName()), p.decl.GetMeta().GetPosition(), []*piece{p}})
		}
		if len(groups) == 0 && g.hasHeader() {
			groups = append(groups, group{path: headerFile})
		}
	}
	g.orderLosses(pieces, func(p *piece) string {
		for _, gr := range groups {
			if slices.Contains(gr.pieces, p) {
				return gr.path
			}
		}
		return ""
	})

	first := ""
	for _, gr := range groups {
		if first == "" || gr.path < first {
			first = gr.path
		}
	}
	var files []*plugin.File
	for _, gr := range groups {
		head := ""
		if gr.path == first {
			head = g.headerComment()
		}
		files = append(files, g.source(pkg, gr.path, gr.pos, head, gr.pieces...))
	}
	return files
}

// piece is one declaration rendered, with the imports it uses.
type piece struct {
	decl    *ir.Decl
	body    string
	imports map[string]string
}

// source assembles pieces into one formatted Go file at path.
// head is the package doc comment, written above the package clause.
func (g *generator) source(pkg, path string, pos *ir.Position, head string, pieces ...*piece) *plugin.File {
	imports := map[string]string{}
	for _, p := range pieces {
		maps.Copy(imports, p.imports)
	}

	var b strings.Builder
	b.WriteString("// Code generated by tdl. DO NOT EDIT.\n\n")
	b.WriteString(head)
	fmt.Fprintf(&b, "package %s\n", pkg)
	writeImports(&b, imports)
	for _, p := range pieces {
		b.WriteString("\n")
		b.WriteString(p.body)
	}

	src, err := format.Source([]byte(b.String()))
	if err != nil {
		// Write the unformatted source so the bug is visible.
		g.Error(pos, "%s is not parseable Go: %v", path, err)
		src = []byte(b.String())
	}
	return &plugin.File{Path: path, Content: src}
}

// piece renders one declaration, or nil for one that generates nothing.
func (g *generator) piece(decl *ir.Decl) (*piece, error) {
	g.strayTags(decl)
	if f, ok := g.foreign[decl]; ok {
		g.foreignLoss(decl, f)
		return nil, nil
	}
	g.declLosses(decl)
	g.imports = map[string]string{}
	g.curClasses = g.paramClasses(decl, true)

	var body strings.Builder
	switch {
	case decl.GetStructure() != nil:
		if err := g.structure(&body, decl); err != nil {
			return nil, err
		}
	case decl.GetEnumeration() != nil:
		if err := g.enumeration(&body, decl); err != nil {
			return nil, err
		}
	case decl.GetNewtype() != nil:
		if err := g.newtype(&body, decl); err != nil {
			return nil, err
		}
	case decl.GetAlias() != nil:
		// An alias is expanded at every use.
		return nil, nil
	case decl.GetClass() != nil:
		if err := g.class(&body, decl); err != nil {
			return nil, err
		}
	case decl.GetUnit() != nil:
		return nil, emit.Lost(emit.LossUnit, decl.GetMeta().GetPosition(),
			"%s is a unit, and units are not generated yet", decl.GetMeta().GetName())
	case decl.GetPrimitive() != nil:
		return nil, nil
	default:
		return nil, nil
	}

	return &piece{decl: decl, body: body.String(), imports: g.imports}, nil
}

// strayTags warns at a `tag` on a declaration or a variant, which has no
// struct tag to set. It is easy to write by accident: in `f => name("F")
// tag("...")`, only the first directive belongs to f.
func (g *generator) strayTags(decl *ir.Decl) {
	stray := func(ds []*ir.Directive, name, kind string) {
		if d, ok := g.Find(ds, "tag"); ok {
			g.Warn(emit.Unsupported(d.GetPosition(),
				"tag sets a field's struct tag, and %s is a %s, so it is ignored; a field takes several directives in a block, as in f { name(...) tag(...) }",
				name, kind))
		}
	}
	name := decl.GetMeta().GetName()
	stray(decl.GetDirectives(), name, "declaration")
	for _, v := range decl.GetEnumeration().GetVariants() {
		stray(v.GetDirectives(), name+"."+v.GetMeta().GetName(), "variant")
	}
}

// structure renders an entity, a value, or a mixin. All three emit the same
// shape; only an entity with a `key` directive gains a Key method.
func (g *generator) structure(b *strings.Builder, decl *ir.Decl) error {
	if err := g.paramProblem(decl); err != nil {
		return err
	}

	name := g.declName(decl)
	g.doc(b, decl.GetMeta())
	g.annotate(b, decl)
	fmt.Fprintf(b, "type %s%s struct {\n", name, g.typeParams(decl))
	if err := g.fields(b, decl.Fields()); err != nil {
		return err
	}
	b.WriteString("}\n")

	keyed := false
	if d, ok := g.Find(decl.GetDirectives(), "key"); ok {
		if err := g.key(b, decl, name, d); err != nil {
			g.lose(emit.LossKey, err)
		} else {
			keyed = true
		}
	}
	if decl.GetStructure().GetKind() == ir.StructKind_STRUCT_KIND_ENTITY && !keyed {
		g.Lose(emit.LossStructKind, decl.GetMeta().GetPosition(), "%s is an entity without a key, and reads back as a value", decl.GetMeta().GetName())
	}
	g.writeValidation(b, decl, -1)
	g.writeMarkers(b, name+typeArgs(decl))
	return nil
}

// key renders an entity's identity: a Key method returning the one field a
// `key` directive names, or a key type holding every field it names. The
// caller warns on error and still emits the entity.
func (g *generator) key(b *strings.Builder, decl *ir.Decl, name string, d *ir.Directive) error {
	fields, err := g.keyFields(decl, name, d)
	if err != nil {
		return err
	}

	types := make([]string, len(fields))
	for i, f := range fields {
		if types[i], err = g.goType(f.GetType()); err != nil {
			return err
		}
	}

	recv, self := receiver(decl, name), name+typeArgs(decl)
	if len(fields) == 1 {
		fmt.Fprintf(b, "\n// Key returns what identifies this %s.\n", name)
		fmt.Fprintf(b, "func (%s %s) Key() %s {\n\treturn %s.%s\n}\n",
			recv, self, types[0], recv, g.fieldName(fields[0]))
		return nil
	}

	// The key type takes the entity's parameters, since its fields may name
	// them.
	keyType := name + "Key"
	fmt.Fprintf(b, "\n// %s is what identifies a %s.\ntype %s%s struct {\n", keyType, name, keyType, g.typeParams(decl))
	inits := make([]string, len(fields))
	for i, f := range fields {
		field := g.fieldName(f)
		fmt.Fprintf(b, "\t%s %s\n", field, types[i])
		inits[i] = fmt.Sprintf("%s: %s.%s", field, recv, field)
	}
	b.WriteString("}\n")
	fmt.Fprintf(b, "\n// Key returns what identifies this %s.\n", name)
	keyType += typeArgs(decl)
	fmt.Fprintf(b, "func (%s %s) Key() %s {\n\treturn %s{%s}\n}\n",
		recv, self, keyType, keyType, strings.Join(inits, ", "))
	return nil
}

// keyFields resolves the fields a `key` directive names, or says why the
// key cannot be generated.
func (g *generator) keyFields(decl *ir.Decl, name string, d *ir.Directive) ([]*ir.Field, error) {
	pos := d.GetPosition()
	at := func(arg *ir.Literal) *ir.Position {
		if p := arg.GetPosition(); p != nil {
			return p
		}
		return pos
	}

	if name == "" {
		return nil, emit.Unsupported(pos, "%s is named \"\" in this target, and a Key method needs a receiver named after it", decl.GetMeta().GetName())
	}
	if decl.GetStructure().GetKind() != ir.StructKind_STRUCT_KIND_ENTITY {
		return nil, emit.Unsupported(pos, "%s has a key, and only an entity is identified by one", name)
	}
	for _, f := range decl.Fields() {
		if g.fieldName(f) == "Key" {
			return nil, emit.Unsupported(pos, "%s has a field named Key, which the Key method would collide with", name)
		}
	}

	var fields []*ir.Field
	for _, arg := range d.GetArgs() {
		if arg.GetKind() != ir.LiteralKind_LITERAL_KIND_NAME {
			return nil, emit.Unsupported(at(arg), "a key names fields, and %q is %s", arg.GetText(), ir.KindName(arg.GetKind()))
		}
		i := slices.IndexFunc(decl.Fields(), func(f *ir.Field) bool { return f.GetMeta().GetName() == arg.GetText() })
		if i < 0 {
			return nil, emit.Unsupported(at(arg), "%s has no field %s to key on", name, arg.GetText())
		}
		f := decl.Fields()[i]
		if slices.Contains(fields, f) {
			return nil, emit.Unsupported(at(arg), "the key of %s names %s twice", name, arg.GetText())
		}
		if !g.comparable(f.GetType()) {
			return nil, emit.Unsupported(at(arg), "a key is compared, and %s.%s is not a comparable Go type", name, arg.GetText())
		}
		fields = append(fields, f)
	}

	if len(fields) > 1 {
		keyType := name + "Key"
		for _, other := range g.Own() {
			if other != decl && g.declName(other) == keyType {
				return nil, emit.Unsupported(pos, "the key type %s would collide with the declaration of that name", keyType)
			}
		}
	}
	return fields, nil
}

// fields renders a struct body.
func (g *generator) fields(b *strings.Builder, fields []*ir.Field) error {
	for _, f := range fields {
		goType, err := g.goType(f.GetType())
		if err != nil {
			return err
		}

		g.doc(b, f.GetMeta())
		fmt.Fprintf(b, "\t%s %s", g.fieldName(f), goType)
		if tag, ok := g.Text(f.GetDirectives(), "tag"); ok {
			fmt.Fprintf(b, " %s", quoteTag(tag))
		}
		b.WriteString("\n")
	}
	return nil
}

// quoteTag writes a struct tag as a raw string, or as an interpreted string
// when it contains a backquote.
func quoteTag(tag string) string {
	if !strings.Contains(tag, "`") {
		return "`" + tag + "`"
	}
	return strconv.Quote(tag)
}

// enumeration renders an enum as one of two shapes, chosen by whether any
// variant carries fields. docs/design/go-backend.md argues the split.
func (g *generator) enumeration(b *strings.Builder, decl *ir.Decl) error {
	name := g.declName(decl)
	variants := decl.GetEnumeration().GetVariants()

	if !emit.Fielded(decl.GetEnumeration()) {
		if len(decl.Params()) > 0 {
			g.Lose(emit.LossGeneric, decl.GetMeta().GetPosition(),
				"%s takes type parameters and none of its variants carries a field, so it is constants, and a Go constant cannot be generic: its parameters are dropped",
				decl.GetMeta().GetName())
		}
		g.doc(b, decl.GetMeta())
		g.annotate(b, decl)
		fmt.Fprintf(b, "type %s string\n\nconst (\n", name)
		for _, v := range variants {
			g.doc(b, v.GetMeta())
			// The value is the variant's name as written.
			fmt.Fprintf(b, "\t%s %s = %q\n",
				g.variantName(name, v), name, v.GetMeta().GetName())
		}
		b.WriteString(")\n")
		g.writeMarkers(b, name)
		return nil
	}

	// A sealed interface: the unexported method keeps the set closed.
	if err := g.paramProblem(decl); err != nil {
		return err
	}

	// The marker takes the enum's parameters, so a ResultOk[int] is not a
	// Result[string].
	sealed := "is" + name + "(" + strings.Join(paramNames(decl), ", ") + ")"
	params, args := g.typeParams(decl), typeArgs(decl)
	g.doc(b, decl.GetMeta())
	g.annotate(b, decl)
	// An interface cannot carry a class's marker, so it embeds the class and
	// each variant carries the marker.
	if embeds := g.markedClasses(); len(embeds) == 0 {
		fmt.Fprintf(b, "type %s%s interface{ %s }\n", name, params, sealed)
	} else {
		fmt.Fprintf(b, "type %s%s interface {\n", name, params)
		for _, e := range embeds {
			fmt.Fprintf(b, "\t%s\n", e)
		}
		fmt.Fprintf(b, "\t%s\n}\n", sealed)
	}

	for i, v := range variants {
		variant := g.variantName(name, v)
		b.WriteString("\n")
		g.doc(b, v.GetMeta())
		fmt.Fprintf(b, "type %s%s struct {\n", variant, params)
		if err := g.fields(b, v.GetFields()); err != nil {
			return err
		}
		b.WriteString("}\n\n")
		fmt.Fprintf(b, "func (%s%s) %s {}\n", variant, args, sealed)
		g.writeMarkers(b, variant+args)
		g.writeValidation(b, decl, i)
	}
	return nil
}

// newtype renders a distinct type over another.
func (g *generator) newtype(b *strings.Builder, decl *ir.Decl) error {
	if err := g.paramProblem(decl); err != nil {
		return err
	}

	base, err := g.goType(decl.GetNewtype().GetBase())
	if err != nil {
		return err
	}
	// Go refuses `type N[T any] T`.
	if slices.Contains(paramNames(decl), base) {
		return emit.Lost(emit.LossGeneric, decl.GetMeta().GetPosition(),
			"%s is a newtype over its type parameter %s, and Go cannot declare a type that is only a type parameter",
			decl.GetMeta().GetName(), base)
	}

	g.doc(b, decl.GetMeta())
	g.annotate(b, decl)
	fmt.Fprintf(b, "type %s%s %s\n", g.declName(decl), g.typeParams(decl), base)
	g.writeValidation(b, decl, -1)
	g.writeMarkers(b, g.declName(decl)+typeArgs(decl))
	return nil
}

// doc writes a node's documentation and deprecation as Go doc comments.
func (g *generator) doc(b *strings.Builder, meta *ir.Meta) {
	lines := emit.Doc(meta)
	for _, line := range lines {
		fmt.Fprintf(b, "// %s\n", line)
	}
	if reason, ok := emit.Deprecated(meta); ok {
		if len(lines) > 0 {
			b.WriteString("//\n")
		}
		if reason == "" {
			reason = defaultReason
		}
		// A reason holding a newline continues the paragraph.
		for i, line := range strings.Split(reason, "\n") {
			if i == 0 {
				line = "Deprecated: " + line
			}
			fmt.Fprintf(b, "// %s\n", line)
		}
	}
}

func (g *generator) use(path string) {
	if _, ok := g.imports[path]; !ok {
		g.imports[path] = ""
	}
}

// useAs records an import the generated code refers to by an alias.
func (g *generator) useAs(path, alias string) {
	g.imports[path] = alias
}

// importNames maps each package the backend imports itself from the name
// generated code uses to its path.
var importNames = map[string]string{
	"errors": "errors",
	"fmt":    "fmt",
	"regexp": "regexp",
	"time":   "time",
	"utf8":   "unicode/utf8",
}

// writeImports writes one import on its own line, several as a sorted block.
func writeImports(b *strings.Builder, imports map[string]string) {
	paths := make([]string, 0, len(imports))
	for p := range imports {
		paths = append(paths, p)
	}
	slices.Sort(paths)

	one := func(p string) string {
		if alias := imports[p]; alias != "" {
			return fmt.Sprintf("%s %q", alias, p)
		}
		return fmt.Sprintf("%q", p)
	}
	switch len(paths) {
	case 0:
	case 1:
		fmt.Fprintf(b, "\nimport %s\n", one(paths[0]))
	default:
		b.WriteString("\nimport (\n")
		for _, p := range paths {
			fmt.Fprintf(b, "\t%s\n", one(p))
		}
		b.WriteString(")\n")
	}
}

// declName is the Go identifier for a declaration, honoring a `name`
// directive.
func (g *generator) declName(decl *ir.Decl) string {
	return g.DeclName(decl, exported)
}

func (g *generator) fieldName(f *ir.Field) string {
	return g.FieldName(f, exported)
}

// variantName is the Go identifier for a variant of the enum named enum.
func (g *generator) variantName(enum string, v *ir.Variant) string {
	return g.VariantName(v, func(s string) string { return enum + exported(s) })
}
