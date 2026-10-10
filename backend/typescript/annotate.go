package typescript

import (
	"cmp"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/unstoppablemango/tdl/ast"
	"github.com/unstoppablemango/tdl/backend/internal/emit"
	"github.com/unstoppablemango/tdl/internal/unlower"
	"github.com/unstoppablemango/tdl/ir"
)

// Generating warns about each fact the TypeScript loses. Under a
// `roundtrip` directive, a declaration that loses any carries itself whole
// as TDL in a JSDoc `@tdl source` tag, and what no declaration carries,
// such as the package and the declarations TypeScript has no form for,
// goes in @tdl tags on an `export {};` at the end of the file.

// lose warns that a fact of the declaration being rendered is lost, which
// makes it carry itself under roundtrip.
func (g *generator) lose(code string, pos *ir.Position, format string, args ...any) {
	g.lost = true
	g.Lose(code, pos, format, args...)
}

// tag is one @tdl tag, its value a Go string literal that cannot end the
// comment it is written in.
func tag(key, value string) string {
	return "@tdl " + key + " " + strings.ReplaceAll(strconv.Quote(value), "*/", `*\u002f`)
}

// docComment writes a top-level node's JSDoc: its doc comment, an
// @deprecated tag, and tags.
func docComment(b *strings.Builder, m *ir.Meta, tags ...string) {
	lines := emit.Doc(m)
	if reason, ok := emit.Deprecated(m); ok {
		if len(lines) > 0 {
			lines = append(lines, "")
		}
		lines = append(lines, strings.TrimSpace("@deprecated "+reason))
	}
	if len(tags) > 0 && len(lines) > 0 {
		lines = append(lines, "")
	}
	emit.BlockComment(b, "", append(lines, tags...))
}

// source is the @tdl tag carrying a declaration whole, when it loses
// anything under roundtrip.
func (g *generator) source(d *ir.Decl) []string {
	if !g.Roundtrip || !g.lost {
		return nil
	}
	for _, decl := range g.Unlowered().Decls {
		if emit.IsDecl(decl) && decl.Name() == d.GetMeta().GetName() {
			return []string{tag("source", emit.PrintItem(decl))}
		}
	}
	return nil
}

// footer is the `export {};` whose JSDoc carries what no declaration
// does: the package when the file name does not say it, the package's doc
// comment, the imports, and every top-level item not emitted, by its
// index; or "".
func (g *generator) footer(path string, emitted map[string]bool) string {
	if !g.Roundtrip {
		return ""
	}
	var tags []string
	if pkg := g.Model.GetPackage(); packageBack(path) != pkg {
		tags = append(tags, tag("package", pkg))
	}
	if doc := g.Model.GetDoc(); len(doc) > 0 {
		tags = append(tags, tag("doc", strings.Join(doc, "\n")))
	}
	for _, imp := range g.Imports() {
		tags = append(tags, tag("import", imp))
	}
	for _, it := range g.Carried(emitted) {
		tags = append(tags, tag("item", fmt.Sprintf("%d %s", it.At, it.Source)))
	}
	if len(tags) == 0 {
		return ""
	}
	var b strings.Builder
	emit.BlockComment(&b, "", tags)
	b.WriteString("export {};\n")
	return b.String()
}

// renamed warns when import would write a `name` directive and the model
// has none, or the reverse: readerWrites is whether the name written
// differs from what import derives.
func (g *generator) renamed(dirs []*ir.Directive, readerWrites bool, format string, args ...any) {
	d, has := g.Find(dirs, "name")
	if has == readerWrites {
		return
	}
	what := fmt.Sprintf(format, args...)
	if has {
		g.lose(emit.LossName, d.GetPosition(), "%s's name directive restates the convention, and does not read back", what)
		return
	}
	g.lose(emit.LossName, nil, "%s reads back with a name directive", what)
}

// docLoss warns when a node's JSDoc does not read back as its doc comment
// and deprecation.
func (g *generator) docLoss(m *ir.Meta) {
	var b strings.Builder
	docComment(&b, m)
	back := &jsdoc{}
	if raw := strings.TrimSuffix(b.String(), "\n"); raw != "" {
		var err error
		if back, err = parseJSDoc(raw); err != nil {
			back = &jsdoc{}
		}
	}
	reason, deprecated := emit.Deprecated(m)
	same := slices.Equal(back.doc, m.GetDoc()) && len(back.tdl) == 0 && len(back.other) == 0 &&
		(back.deprecated != nil) == deprecated && (back.deprecated == nil || back.deprecated.Reason == reason)
	if !same {
		g.lose(emit.LossDoc, m.GetPosition(), "%s's doc comment does not read back from its JSDoc", m.GetName())
	}
}

// declLosses warns about what a declaration loses besides its fields:
// its name, its doc comment, its kind, and its conformances.
func (g *generator) declLosses(d *ir.Decl, written string) {
	name, pos := d.GetMeta().GetName(), d.GetMeta().GetPosition()
	if written != name {
		g.lose(emit.LossName, pos, "%s is written %s, which reads back as its name", name, written)
	} else {
		g.renamed(d.GetDirectives(), emit.Pascal(written) != written, "%s", name)
	}
	g.docLoss(d.GetMeta())

	conforms := d.GetEnumeration().GetConforms()
	if s := d.GetStructure(); s != nil {
		conforms = s.GetConforms()
		switch s.GetKind() {
		case ir.StructKind_STRUCT_KIND_ENTITY:
			g.lose(emit.LossStructKind, pos, "%s is an entity, and reads back as a value", name)
		case ir.StructKind_STRUCT_KIND_MIXIN:
			g.lose(emit.LossStructKind, pos, "%s is a mixin, and reads back as a value", name)
		}
	}
	for _, c := range conforms {
		if n := c.GetClass().GetName(); d.GetStructure() == nil || n != "Entity" || c.GetExtern() != nil || len(c.GetArgs()) > 0 {
			g.lose(emit.LossClass, c.GetPosition(), "%s conforms to %s, which TypeScript does not carry", name, n)
		}
	}
}

// includeLosses warns about each include, flattened into the fields it
// copies.
func (g *generator) includeLosses(owner string, fields []*ir.Field) {
	includes := unlower.Includes(g.Model, fields)
	for i, inc := range includes {
		if inc != "" && (i == 0 || includes[i-1] != inc) {
			g.lose(emit.LossInclude, fields[i].GetMeta().GetPosition(), "%s's include of %s is flattened into its fields", owner, inc)
		}
	}
}

// fieldLosses warns about what a property loses, and makes the
// declaration carry itself when the property does not read back as the
// field.
func (g *generator) fieldLosses(owner string, f *ir.Field, ref *emit.Ref) {
	name, pos := f.GetMeta().GetName(), f.GetMeta().GetPosition()
	written := g.FieldName(f, asWritten)
	back := written
	if !tdlIdent.MatchString(written) {
		back = emit.Camel(written)
	}
	if back != name {
		g.lose(emit.LossName, pos, "%s.%s is written %s, which reads back as %s", owner, name, written, back)
	} else {
		g.renamed(f.GetDirectives(), back != written, "%s.%s", owner, name)
	}
	if f.GetOwned() {
		g.lose(emit.LossOwned, pos, "%s.%s is owned, which TypeScript does not carry", owner, name)
	}
	if f.GetDefaultValue() != nil {
		g.lose(emit.LossDefault, pos, "%s.%s has a default, which is not written", owner, name)
	}
	g.typeLosses(f.GetType(), pos, modeProperty)
	g.docLoss(f.GetMeta())

	typ, cs, narrowed := g.back(ref, f.GetConstraints(), modeProperty)
	got := &ast.Field{DeclHead: ast.DeclHead{N: back}, Type: typ, Constraints: cs}
	want := unlower.Field(g.Model, f)
	want.Doc, want.DocP, want.Dep = nil, nil, nil
	if narrowed && g.allEnforced(f.GetConstraints()) && constraintsText(cs) != constraintsText(want.Constraints) {
		g.lose(emit.LossConstraint, pos, "%s.%s is narrowed to literals, which read back as oneOf(...)", owner, name)
	}
	if ast.PrintField(got) != ast.PrintField(want) {
		g.lost = true
	}
}

// newtypeLosses warns about what a newtype's alias loses, and makes it
// carry itself when the alias does not read back as the newtype.
func (g *generator) newtypeLosses(d *ir.Decl, ref *emit.Ref) {
	name, pos := d.GetMeta().GetName(), d.GetMeta().GetPosition()
	g.typeLosses(d.GetNewtype().GetBase(), nil, modeAlias)

	typ, cs, narrowed := g.back(ref, d.GetNewtype().GetValueConstraints(), modeAlias)
	if narrowed && len(cs) == 1 && typ.N == "string" && !slices.ContainsFunc(cs[0].Args, func(l *ast.Literal) bool { return !tdlName(l.Text) }) {
		g.lose(emit.LossNewtype, pos, "newtype %s is a union of string literals, which reads back as an enum", name)
		return
	}
	var want *ast.NewtypeDecl
	for _, decl := range g.Unlowered().Decls {
		if n, ok := decl.(*ast.NewtypeDecl); ok && n.N == name {
			want = n
		}
	}
	if want == nil {
		g.lost = true
		return
	}
	if narrowed && g.allEnforced(d.GetNewtype().GetValueConstraints()) && constraintsText(cs) != constraintsText(want.Constraints) {
		g.lose(emit.LossConstraint, pos, "newtype %s is narrowed to literals, which read back as oneOf(...)", name)
	}
	got := &ast.Field{DeclHead: ast.DeclHead{N: "x"}, Type: typ, Constraints: cs}
	if ast.PrintField(got) != ast.PrintField(&ast.Field{DeclHead: ast.DeclHead{N: "x"}, Type: want.Base, Constraints: want.Constraints}) {
		g.lost = true
	}
}

// allEnforced reports whether a literal union enforces every constraint,
// since [emit.Session.WarnUnenforced] warns about the rest.
func (g *generator) allEnforced(cs []*ir.Constraint) bool {
	return !slices.ContainsFunc(cs, func(c *ir.Constraint) bool { return !g.enforced[c] })
}

func constraintsText(cs []*ast.Constraint) string {
	return ast.PrintField(&ast.Field{DeclHead: ast.DeclHead{N: "x"}, Type: &ast.TypeRef{N: "T"}, Constraints: cs})
}

// mode is where a type is written, which decides how `?` and `| null`
// read back.
type mode int

const (
	// modeProperty is a property's type: `?` is T?, and `| null` is
	// T | null.
	modeProperty mode = iota
	// modeAlias is a newtype's alias: `| null` is T | null.
	modeAlias
	// modeElem is a collection's element or a map's value: `| null` is T?.
	modeElem
	// modeKey is a map's key.
	modeKey
)

// optionLike reports whether a type is Option or Nullable, and which.
func (g *generator) optionLike(t *ir.Type) (string, bool) {
	d := g.Model.Decl(t.GetCtor())
	if d == nil || d.GetEnumeration() == nil || len(t.GetArgs()) != 1 {
		return "", false
	}
	n := d.GetMeta().GetName()
	return n, n == "Option" || n == "Nullable"
}

// typeLosses warns about what a type loses in TypeScript: a primitive's
// kind, a set, sugar import writes another way, and absence import reads
// another way. A newtype or alias it names warns where it is declared.
func (g *generator) typeLosses(id *ir.ID, at *ir.Position, m mode) {
	t := g.Model.Type(id)
	if t == nil || t.GetParam() != nil || t.GetUnit() != nil {
		return
	}
	pos := cmp.Or(at, t.GetPosition())

	if _, ok := g.optionLike(t); ok {
		var chain []string
		cur := id
		for {
			ct := g.Model.Type(cur)
			n, ok := g.optionLike(ct)
			if !ok {
				break
			}
			wrote := ct.GetWrote()
			switch {
			case n == "Option" && wrote == ir.SyntacticForm_SYNTACTIC_FORM_QUESTION:
				chain = append(chain, "?")
			case n == "Nullable" && wrote == ir.SyntacticForm_SYNTACTIC_FORM_OR_NULL:
				chain = append(chain, "null")
			default:
				chain = append(chain, n)
			}
			cur = ct.GetArgs()[0]
		}
		allowed := map[mode][]string{
			modeProperty: {"?", "null", "null ?"},
			modeAlias:    {"null"},
			modeElem:     {"?"},
		}[m]
		if !slices.Contains(allowed, strings.Join(chain, " ")) {
			g.lose(emit.LossOptional, pos, "%s reads back as %s", describeChain(chain), map[mode]string{
				modeProperty: "T?, T | null, or T? | null",
				modeAlias:    "T | null",
				modeElem:     "an optional element, T?",
				modeKey:      "a key that is always present",
			}[m])
		}
		g.typeLosses(cur, at, m)
		return
	}

	decl := g.Model.Decl(t.GetCtor())
	if decl == nil || decl.GetAlias() != nil || decl.GetNewtype() != nil || decl.GetPrimitive() == nil {
		return
	}
	name, wrote := decl.GetMeta().GetName(), t.GetWrote()
	args := t.GetArgs()
	switch {
	case name == "Set":
		g.lose(emit.LossCollection, pos, "a set is a TypeScript array, and reads back as a list")
	case name == "List" && wrote != ir.SyntacticForm_SYNTACTIC_FORM_BRACKETS,
		name == "Map" && wrote != ir.SyntacticForm_SYNTACTIC_FORM_ARROW:
		g.lose(emit.LossCollection, pos, "%s reads back written as sugar", name)
	case scalars[name] == "string" && name != "string":
		g.lose(emit.LossPrimitive, pos, "%s is a TypeScript string, which reads back as string", name)
	case scalars[name] == "number" && name != "float64":
		g.lose(emit.LossPrimitive, pos, "%s is a TypeScript number, which reads back as float64", name)
	}
	for i, arg := range args {
		inner := modeElem
		if name == "Map" && i == 0 {
			inner = modeKey
		}
		g.typeLosses(arg, at, inner)
	}
}

func describeChain(chain []string) string {
	var parts []string
	for _, c := range chain {
		switch c {
		case "?":
			parts = append(parts, "T?")
		case "null":
			parts = append(parts, "T | null")
		default:
			parts = append(parts, c)
		}
	}
	return strings.Join(parts, " around ")
}

// back is the TDL type, and constraints, a field or newtype of type r
// with constraints cs reads back as, and whether its constraints narrowed
// it to literals.
func (g *generator) back(r *emit.Ref, cs []*ir.Constraint, m mode) (*ast.TypeRef, []*ast.Constraint, bool) {
	optional, nullable := false, false
	if m == modeProperty || m == modeAlias {
		for r.Form == emit.Option || r.Form == emit.Nullable {
			if r.Form == emit.Option && m == modeProperty {
				optional = true
			} else {
				nullable = true
			}
			r = r.Elem
		}
	}

	var inherited []*ir.Constraint
	if m == modeProperty {
		inherited = r.Decl.GetNewtype().GetValueConstraints()
	}
	if lits := g.narrow(r, cs, inherited); lits != "" {
		typ, oneOf := literalsBack(lits)
		typ.Optional, typ.Nullable = optional, nullable
		return typ, []*ast.Constraint{oneOf}, true
	}
	typ := g.backType(r, m)
	typ.Optional = typ.Optional || optional
	typ.Nullable = typ.Nullable || nullable
	return typ, nil, false
}

// literalsBack reads a literal union back, as import does.
func literalsBack(lits string) (*ast.TypeRef, *ast.Constraint) {
	oneOf := &ast.Constraint{N: "oneOf"}
	str := strings.HasPrefix(lits, `"`)
	for v := range strings.SplitSeq(lits, " | ") {
		if str {
			s, _ := strconv.Unquote(v)
			oneOf.Args = append(oneOf.Args, &ast.Literal{Kind: ast.LitString, Text: s})
			continue
		}
		kind := ast.LitInt
		if strings.ContainsAny(v, ".eE") {
			kind = ast.LitFloat
		}
		oneOf.Args = append(oneOf.Args, &ast.Literal{Kind: kind, Text: v})
	}
	if str {
		return &ast.TypeRef{N: "string"}, oneOf
	}
	return &ast.TypeRef{N: "float64"}, oneOf
}

// backType is the TDL type a reference's TypeScript reads back as.
func (g *generator) backType(r *emit.Ref, m mode) *ast.TypeRef {
	switch r.Form {
	case emit.Prim:
		switch scalars[r.Name] {
		case "number":
			return &ast.TypeRef{N: "float64"}
		case "boolean":
			return &ast.TypeRef{N: "bool"}
		}
		return &ast.TypeRef{N: "string"}
	case emit.List, emit.Set:
		return &ast.TypeRef{List: g.backType(r.Elem, modeElem)}
	case emit.Map:
		return &ast.TypeRef{MapKey: g.backType(r.Key, modeKey), MapValue: g.backType(r.Elem, modeElem)}
	case emit.Option, emit.Nullable:
		inner := g.backType(r.Elem, m)
		if m == modeElem || m == modeKey {
			inner.Optional = true
		} else {
			inner.Nullable = true
		}
		return inner
	}
	return &ast.TypeRef{N: r.Decl.GetMeta().GetName()}
}
