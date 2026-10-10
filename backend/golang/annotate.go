package golang

import (
	"errors"
	"fmt"
	goast "go/ast"
	"go/format"
	goparser "go/parser"
	"go/token"
	"slices"
	"strconv"
	"strings"

	"github.com/unstoppablemango/tdl/ast"
	"github.com/unstoppablemango/tdl/backend/internal/emit"
	"github.com/unstoppablemango/tdl/internal/unlower"
	"github.com/unstoppablemango/tdl/ir"
	"github.com/unstoppablemango/tdl/plugin"
)

// Under a `roundtrip` directive, what import would not read back from a
// declaration is carried on it as //tdl: comment directives, which a doc
// comment's text leaves out:
//
//   - source: the declaration as TDL, when import would read it otherwise;
//   - at: its index among the model's items, when the files' order is not
//     the model's.
//
// What the package says, on the first file's package clause:
//
//   - package: the TDL package, when the package clause is not it;
//   - doc: the package's doc comment, when the Go one does not read back;
//   - import: each TDL import;
//   - item: an index and a top-level item no Go type declares.
//
// Which declarations need a source is decided by reading the files back
// without annotations, so each is carried exactly when import would miss
// something.

// headerFile holds the package's annotations when no declaration has a
// file to hold them.
const headerFile = "tdl.go"

// lose warns about an error the way [emit.Session.Lose] warns about a fact,
// giving it code when it carries none.
func (g *generator) lose(code string, err error) {
	if g.Roundtrip {
		return
	}
	if u, ok := errors.AsType[*emit.UnsupportedError](err); ok && u.Code == "" {
		u.Code = code
	}
	g.Warn(err)
}

// annotate writes a declaration's //tdl: directives, ending its doc comment.
func (g *generator) annotate(b *strings.Builder, decl *ir.Decl) {
	writeAnns(b, g.notes[decl])
}

func writeAnns(b *strings.Builder, anns []kv) {
	for _, a := range anns {
		value := a.value
		if a.key != "at" {
			value = strconv.Quote(value)
		}
		fmt.Fprintf(b, "%s%s %s\n", annotation, a.key, value)
	}
}

// hasHeader reports whether the package clause carries annotations, which
// need a file even when no declaration has one.
func (g *generator) hasHeader() bool {
	return g.Roundtrip && len(g.header) > 0
}

// headerComment is the package's doc comment and annotations.
func (g *generator) headerComment() string {
	var b strings.Builder
	g.doc(&b, &ir.Meta{Doc: g.Model.GetDoc()})
	writeAnns(&b, g.header)
	return b.String()
}

// simulate reads the files back as import would without annotations, and
// decides what the annotations carry.
func (g *generator) simulate(files []*plugin.File, pkg string) {
	r := newReader()
	read, err := r.read(files)

	g.notes = map[*ir.Decl][]kv{}
	emitted := map[string]bool{}
	for _, p := range g.emitted {
		name := p.decl.GetMeta().GetName()
		emitted[name] = true
		at := g.Index(name)
		want := emit.PrintItem(g.Unlowered().Decls[at])
		got := ""
		if rd := r.decls[g.declName(p.decl)]; err == nil && rd != nil {
			got = emit.PrintItem(rd)
		}
		if got != want {
			g.notes[p.decl] = append(g.notes[p.decl], kv{"source", want})
		}
		if g.reordered {
			g.notes[p.decl] = append(g.notes[p.decl], kv{"at", strconv.Itoa(at)})
		}
	}

	back, doc := pkg, []string(nil)
	if back == "main" {
		back = ""
	}
	if err == nil && read.Package != nil {
		doc = read.Package.Doc
	}
	g.header = nil
	if back != g.Model.GetPackage() {
		g.header = append(g.header, kv{"package", g.Model.GetPackage()})
	}
	if !slices.Equal(doc, g.Model.GetDoc()) {
		g.header = append(g.header, kv{"doc", strings.Join(g.Model.GetDoc(), "\n")})
	}
	for _, imp := range g.Imports() {
		g.header = append(g.header, kv{"import", imp})
	}
	for _, it := range g.Carried(emitted) {
		g.header = append(g.header, kv{"item", fmt.Sprintf("%d %s", it.At, it.Source)})
	}
}

// orderLosses decides whether import reads the declarations in another
// order than the model's: files in path order, and each file in its own.
func (g *generator) orderLosses(pieces []*piece, path func(*piece) string) {
	read := slices.Clone(pieces)
	slices.SortStableFunc(read, func(a, b *piece) int { return strings.Compare(path(a), path(b)) })
	g.reordered = !slices.Equal(read, pieces)
	if g.reordered {
		g.Lose(emit.LossOrder, nil, "the files read back in path order, which is not the model's")
	}
}

// packageLosses warns when the package clause or the package's doc
// comment does not read back.
func (g *generator) packageLosses(pkg string, pos *ir.Position) {
	if g.Roundtrip {
		return
	}
	back := pkg
	if back == "main" {
		back = ""
	}
	if want := g.Model.GetPackage(); back != want {
		g.Lose(emit.LossName, pos, "package %s is written package %s, which reads back as package %q", want, pkg, back)
	}
	if doc := g.Model.GetDoc(); len(doc) > 0 {
		if lines, _, ok := g.docBack(&ir.Meta{Doc: doc}, sitePackage); !ok || !slices.Equal(lines, doc) {
			g.Lose(emit.LossDoc, nil, "the package's doc comment does not read back from a Go comment")
		}
	}
}

// foreignLoss warns when a declaration mapped to another package's type is
// not what import reads for one: an empty struct named for the type.
func (g *generator) foreignLoss(d *ir.Decl, f foreignType) {
	if g.Roundtrip {
		return
	}
	name := d.GetMeta().GetName()
	want := emit.PrintItem(&ast.StructDecl{DeclHead: ast.DeclHead{N: f.name}, Keyword: "type"})
	if i := g.Index(name); i < 0 || emit.PrintItem(g.Unlowered().Decls[i]) != want {
		g.Lose(emit.LossUnsupported, d.GetMeta().GetPosition(), "%s is %s, and reads back as an empty declaration named %s", name, f.ref(), f.name)
	}
}

// declLosses warns about what a declaration loses that rendering it does
// not report itself.
func (g *generator) declLosses(d *ir.Decl) {
	if g.Roundtrip {
		return
	}
	pos, name := d.GetMeta().GetPosition(), d.GetMeta().GetName()
	switch {
	case d.GetAlias() != nil:
		g.Lose(emit.LossAlias, pos, "alias %s is expanded where used", name)
		if len(d.Params()) == 0 {
			g.typeLosses(d.GetAlias().GetTarget(), nil)
		}
		return
	case d.GetStructure() == nil && d.GetEnumeration() == nil && d.GetNewtype() == nil && d.GetClass() == nil:
		return
	}

	goName := g.declName(d)
	if goName != name {
		g.Lose(emit.LossName, pos, "%s is written %s, which reads back as its name", name, goName)
	} else {
		g.CheckRenamed(d.GetDirectives(), false, "%s", name)
	}
	g.docLoss(d.GetMeta(), siteType)
	g.conformsLoss(d)

	switch {
	case d.GetStructure() != nil:
		if d.GetStructure().GetKind() == ir.StructKind_STRUCT_KIND_MIXIN {
			g.Lose(emit.LossStructKind, pos, "%s is a mixin, and reads back as a value", name)
		}
		g.fieldsLosses(name, d.Fields(), true)
	case d.GetEnumeration() != nil:
		g.variantLosses(d, goName)
	case d.GetNewtype() != nil:
		g.typeLosses(d.GetNewtype().GetBase(), nil)
	case d.GetClass() != nil:
		if len(d.GetClass().GetFields()) > 0 {
			g.Lose(emit.LossClass, pos, "%s's fields are not written, since generated code never reads them", name)
		}
	}
}

func (g *generator) variantLosses(d *ir.Decl, goName string) {
	name := d.GetMeta().GetName()
	e := d.GetEnumeration()
	for _, v := range e.GetVariants() {
		vname, vpos := v.GetMeta().GetName(), v.GetMeta().GetPosition()
		ident := g.variantName(goName, v)
		if !emit.Fielded(e) {
			// A constant's value is the variant's name.
			g.CheckRenamed(v.GetDirectives(), ident != goName+exported(vname), "%s.%s", name, vname)
			g.docLoss(v.GetMeta(), siteConst)
			continue
		}
		back, ok := strings.CutPrefix(ident, goName)
		switch {
		case !ok || back == "":
			g.Lose(emit.LossName, vpos, "%s.%s is written %s, which is not named for %s, so %s reads back as a class", name, vname, ident, goName, name)
		case back != vname:
			g.Lose(emit.LossName, vpos, "%s.%s is written %s, which reads back as %s", name, vname, ident, back)
		default:
			g.CheckRenamed(v.GetDirectives(), ident != goName+exported(back), "%s.%s", name, vname)
		}
		g.docLoss(v.GetMeta(), siteType)
		g.fieldsLosses(name+"."+vname, v.GetFields(), false)
	}
}

// conformsLoss warns when the classes whose markers a declaration carries
// are not the ones it conforms to, past Entity, which its key says.
func (g *generator) conformsLoss(d *ir.Decl) {
	conforms := d.GetStructure().GetConforms()
	if e := d.GetEnumeration(); e != nil {
		conforms = e.GetConforms()
	}
	var want []string
	for _, c := range conforms {
		if ext := c.GetExtern(); ext != nil {
			want = append(want, ext.GetName())
			continue
		}
		n := g.Model.Decl(c.GetClass()).GetMeta().GetName()
		if d.GetStructure() == nil || n != "Entity" || len(c.GetArgs()) > 0 {
			want = append(want, n)
		}
	}
	var back []string
	for _, c := range g.marks[g.cur] {
		back = append(back, g.Model.GetDecls()[c].GetMeta().GetName())
	}
	if !slices.Equal(want, back) {
		g.Lose(emit.LossClass, d.GetMeta().GetPosition(), "%s conforms to %s, and its markers read back as %s",
			d.GetMeta().GetName(), list(want), list(back))
	}
}

func list(names []string) string {
	if len(names) == 0 {
		return "none"
	}
	return strings.Join(names, ", ")
}

// fieldsLosses warns about what each field loses, and about an include
// flattened into its fields.
func (g *generator) fieldsLosses(owner string, fields []*ir.Field, includes bool) {
	incs := make([]string, len(fields))
	if includes {
		incs = unlower.Includes(g.Model, fields)
	}
	for i, f := range fields {
		if inc := incs[i]; inc != "" && (i == 0 || incs[i-1] != inc) {
			g.Lose(emit.LossInclude, f.GetMeta().GetPosition(), "%s's include of %s is flattened into its fields", owner, inc)
		}
		g.fieldLoss(owner, f)
	}
}

func (g *generator) fieldLoss(owner string, f *ir.Field) {
	name, pos := f.GetMeta().GetName(), f.GetMeta().GetPosition()
	goName := g.fieldName(f)
	if back := unexport(goName); back != name {
		g.Lose(emit.LossName, pos, "%s.%s is written %s, which reads back as %s", owner, name, goName, back)
	} else {
		g.CheckRenamed(f.GetDirectives(), exported(back) != goName, "%s.%s", owner, name)
	}
	if f.GetOwned() {
		g.Lose(emit.LossOwned, pos, "%s.%s is owned, which Go does not carry", owner, name)
	}
	if f.GetDefaultValue() != nil {
		g.Lose(emit.LossDefault, pos, "%s.%s has a default, which is not written", owner, name)
	}
	g.typeLosses(f.GetType(), pos)
	g.docLoss(f.GetMeta(), siteField)
}

// lossyPrimitives maps each primitive whose Go type reads back as another
// to that one.
var lossyPrimitives = map[string]string{
	"int":     "int64",
	"uuid":    "string",
	"decimal": "string",
	"date":    "instant",
}

// typeLosses warns about what a type loses in Go: a primitive's kind, and
// sugar import writes another way. A newtype or alias it names warns where
// it is declared.
func (g *generator) typeLosses(id *ir.ID, at *ir.Position) {
	t := g.Model.Type(id)
	if t == nil || t.GetParam() != nil || t.GetUnit() != nil {
		return
	}
	decl := g.Model.Decl(t.GetCtor())
	if decl == nil || decl.GetAlias() != nil || decl.GetNewtype() != nil {
		return
	}
	// A type is interned at its first use, so a field's own position says
	// which use is meant.
	pos := at
	if pos == nil {
		pos = t.GetPosition()
	}
	name, wrote := decl.GetMeta().GetName(), t.GetWrote()
	switch {
	case decl.GetPrimitive() != nil:
		switch {
		case name == "List" && wrote != ir.SyntacticForm_SYNTACTIC_FORM_BRACKETS,
			name == "Set" && wrote != ir.SyntacticForm_SYNTACTIC_FORM_BRACES,
			name == "Map" && wrote != ir.SyntacticForm_SYNTACTIC_FORM_ARROW:
			g.Lose(emit.LossCollection, pos, "%s reads back written as sugar", name)
		case lossyPrimitives[name] != "":
			g.Lose(emit.LossPrimitive, pos, "%s is Go %s, which reads back as %s", name, primitives[name], lossyPrimitives[name])
		}
	case decl.GetEnumeration() != nil && name == "Nullable":
		g.Lose(emit.LossOptional, pos, "a nullable value is a Go pointer, and reads back as T?")
	case decl.GetEnumeration() != nil && name == "Option" && wrote != ir.SyntacticForm_SYNTACTIC_FORM_QUESTION:
		g.Lose(emit.LossOptional, pos, "Option reads back written as T?")
	}
	for _, arg := range t.GetArgs() {
		g.typeLosses(arg, at)
	}
}

// site is where a doc comment is written, which decides how gofmt
// formats it.
type site int

const (
	sitePackage site = iota
	siteType
	siteField
	siteConst
)

// docLoss warns when a node's doc comment or deprecation does not read
// back from the Go comment doc writes.
func (g *generator) docLoss(meta *ir.Meta, at site) {
	_, deprecated := emit.Deprecated(meta)
	if len(meta.GetDoc()) == 0 && !deprecated {
		return
	}
	lines, dep, ok := g.docBack(meta, at)
	reason, _ := emit.Deprecated(meta)
	if ok && slices.Equal(lines, meta.GetDoc()) && (dep != nil) == deprecated && (dep == nil || dep.Reason == reason) {
		return
	}
	g.Lose(emit.LossDoc, meta.GetPosition(), "%s's doc comment does not read back from a Go comment", meta.GetName())
}

// docBack is the doc comment and deprecation import reads from what doc
// writes for a node, formatted as gofmt formats it where it is written.
func (g *generator) docBack(meta *ir.Meta, at site) ([]string, *ast.Deprecation, bool) {
	var c strings.Builder
	g.doc(&c, meta)
	var src string
	switch at {
	case sitePackage:
		src = c.String() + "package p\n"
	case siteType:
		src = "package p\n\n" + c.String() + "type T int\n"
	case siteField:
		src = "package p\n\ntype T struct {\n" + c.String() + "F int\n}\n"
	case siteConst:
		src = "package p\n\nconst (\n" + c.String() + "C = 1\n)\n"
	}
	formatted, err := format.Source([]byte(src))
	if err != nil {
		return nil, nil, false
	}
	f, err := goparser.ParseFile(token.NewFileSet(), "", formatted, goparser.ParseComments)
	if err != nil {
		return nil, nil, false
	}
	var cg *goast.CommentGroup
	switch at {
	case sitePackage:
		cg = f.Doc
	case siteType:
		cg = f.Decls[0].(*goast.GenDecl).Doc
	case siteField:
		cg = f.Decls[0].(*goast.GenDecl).Specs[0].(*goast.TypeSpec).Type.(*goast.StructType).Fields.List[0].Doc
	case siteConst:
		cg = f.Decls[0].(*goast.GenDecl).Specs[0].(*goast.ValueSpec).Doc
	}
	lines, dep := readDoc(cg, at != sitePackage)
	return lines, dep, true
}
