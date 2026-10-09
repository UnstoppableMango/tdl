// Package unlower turns a resolved model back into a parse tree, the
// reverse of internal/sema. [ast.Fprint] prints the result, so a model a
// reverse backend builds becomes TDL source through one printer.
//
// Lowering the printed file gives back the model, positions aside, for any
// model lowering produced. docs/design/reverse.md describes where it is used.
package unlower

import (
	"cmp"
	"slices"
	"strconv"
	"strings"

	"github.com/unstoppablemango/tdl/ast"
	"github.com/unstoppablemango/tdl/ir"
	"github.com/unstoppablemango/tdl/parser"
	"github.com/unstoppablemango/tdl/prelude"
)

// File rebuilds the source of a model: its package, imports, own
// declarations, instances, and target blocks. Prelude declarations are
// left out, since lowering loads the prelude beneath every file.
func File(m *ir.Model) *ast.File {
	u := newUnlowerer(m)
	f := &ast.File{}
	if m.GetPackage() != "" || len(m.GetDoc()) > 0 {
		f.Package = &ast.PackageDecl{Doc: m.GetDoc(), DocP: docPositions(m.GetDoc()), Path: m.GetPackage()}
	}
	for _, imp := range m.GetImports() {
		f.Imports = append(f.Imports, &ast.ImportDecl{Path: imp.GetPath(), Alias: imp.GetAlias()})
	}
	f.Decls = u.decls()
	return f
}

// Field rebuilds one field of a model's declaration, as it would be
// written in the file [File] rebuilds.
func Field(m *ir.Model, f *ir.Field) *ast.Field {
	return newUnlowerer(m).field(f)
}

// Includes names, for each field of a struct, the mixin whose `include`
// [File] writes for it, or "" for a field written in place.
func Includes(m *ir.Model, fields []*ir.Field) []string {
	u := newUnlowerer(m)
	out := make([]string, len(fields))
	var included []int
	for i, f := range fields {
		if f.GetIncludedFrom() != nil {
			included = append(included, i)
		}
	}
	for len(included) > 0 {
		run := make([]*ir.Field, len(included))
		for i, idx := range included {
			run[i] = fields[idx]
		}
		name, n := u.include(run)
		for _, idx := range included[:n] {
			out[idx] = name
		}
		included = included[n:]
	}
	return out
}

func newUnlowerer(m *ir.Model) *unlowerer {
	u := &unlowerer{model: m, aliases: map[string]string{}}
	for _, imp := range m.GetImports() {
		if _, seen := u.aliases[imp.GetPackage()]; !seen && imp.GetAlias() != "_" {
			u.aliases[imp.GetPackage()] = imp.GetAlias()
		}
	}
	return u
}

type unlowerer struct {
	model   *ir.Model
	aliases map[string]string // package to the alias importing it; absent for a `_` import
}

// own reports whether a declaration is the model's rather than the
// prelude's, by the rule backend/internal/emit.IsOwn states.
func own(d *ir.Decl) bool {
	return d.GetMeta().GetPosition().GetFilename() != prelude.Name
}

// ordered is a top-level declaration and the position lowering recorded
// for it among the file's declarations.
type ordered struct {
	order int32
	decl  ast.Decl
}

// decls returns the file's declarations in source order. Declarations and
// instances record their index among the file's declarations; target blocks
// record their index among target blocks, so they fill the gaps.
func (u *unlowerer) decls() []ast.Decl {
	var items []ordered
	for _, d := range u.model.GetDecls() {
		if own(d) {
			items = append(items, ordered{d.GetMeta().GetOrder(), u.decl(d)})
		}
	}
	for _, inst := range u.model.GetInstances() {
		items = append(items, ordered{inst.GetMeta().GetOrder(), u.instance(inst)})
	}
	slices.SortStableFunc(items, func(a, b ordered) int { return cmp.Compare(a.order, b.order) })

	targets := u.targets()
	out := make([]ast.Decl, 0, len(items)+len(targets))
	for len(items) > 0 || len(targets) > 0 {
		next := int32(len(out))
		if len(items) > 0 && (items[0].order <= next || len(targets) == 0) {
			out = append(out, items[0].decl)
			items = items[1:]
			continue
		}
		out = append(out, targets[0])
		targets = targets[1:]
	}
	return out
}

func (u *unlowerer) decl(d *ir.Decl) ast.Decl {
	head := declHead(d.GetMeta())
	switch {
	case d.GetPrimitive() != nil:
		return &ast.PrimitiveDecl{DeclHead: head, Kind: kind(d.GetPrimitive().GetKind())}

	case d.GetAlias() != nil:
		a := d.GetAlias()
		return &ast.AliasDecl{DeclHead: head, Params: params(a.GetParams()), Target: u.typeRef(a.GetTarget())}

	case d.GetNewtype() != nil:
		n := d.GetNewtype()
		var written []*ir.Constraint
		for _, c := range n.GetValueConstraints() {
			if c.GetFrom() == nil { // inherited ones are lowering's to add
				written = append(written, c)
			}
		}
		return &ast.NewtypeDecl{
			DeclHead:    head,
			Params:      params(n.GetParams()),
			Base:        u.typeRef(n.GetBase()),
			Requires:    u.classRefs(n.GetConstraints()),
			Constraints: u.constraints(written),
		}

	case d.GetStructure() != nil:
		s := d.GetStructure()
		keyword := "type"
		if s.GetKind() == ir.StructKind_STRUCT_KIND_MIXIN {
			keyword = "mixin"
		}
		return &ast.StructDecl{
			DeclHead: head,
			Keyword:  keyword,
			Params:   params(s.GetParams()),
			Conforms: u.classRefs(s.GetConforms()),
			Requires: u.classRefs(s.GetConstraints()),
			Members:  u.members(s.GetFields()),
		}

	case d.GetEnumeration() != nil:
		e := d.GetEnumeration()
		out := &ast.EnumDecl{
			DeclHead: head,
			Params:   params(e.GetParams()),
			Conforms: u.classRefs(e.GetConforms()),
			Requires: u.classRefs(e.GetConstraints()),
		}
		for _, v := range e.GetVariants() {
			variant := &ast.Variant{DeclHead: declHead(v.GetMeta())}
			for _, f := range v.GetFields() {
				variant.Fields = append(variant.Fields, u.field(f))
			}
			if len(variant.Fields) > 0 {
				// The printer expands a payload only for a variant that ends
				// somewhere, which is what lets a documented field keep its
				// doc comment.
				variant.End = ast.Position{Line: 1}
			}
			out.Variants = append(out.Variants, variant)
		}
		return out

	case d.GetClass() != nil:
		c := d.GetClass()
		out := &ast.ClassDecl{
			DeclHead: head,
			Params:   params(c.GetParams()),
			Conforms: u.classRefs(c.GetRequiresClasses()),
			Requires: u.classRefs(c.GetConstraints()),
		}
		for _, dep := range c.GetFunDeps() {
			out.FunDeps = append(out.FunDeps, &ast.FunDep{From: dep.GetFrom(), To: dep.GetTo()})
		}
		for _, at := range c.GetAssocTypes() {
			out.Members = append(out.Members, &ast.AssocTypeReq{DeclHead: declHead(at.GetMeta()), Kind: kind(at.GetKind())})
		}
		for _, f := range c.GetFields() {
			out.Members = append(out.Members, u.field(f))
		}
		return out

	case d.GetUnit() != nil:
		def := d.GetUnit()
		out := &ast.UnitDecl{DeclHead: head}
		if !def.GetBase() {
			out.Expr = u.unitExpr(def.GetUnit())
		}
		return out
	}

	// A declaration with no node, which lowering never produces, keeps its
	// name as an opaque primitive.
	return &ast.PrimitiveDecl{DeclHead: head}
}

// members rebuilds a struct body: the fields it wrote, then an `include`
// for each run of fields copied from a mixin.
func (u *unlowerer) members(fields []*ir.Field) []ast.Member {
	var written, included []*ir.Field
	for _, f := range fields {
		if f.GetIncludedFrom() == nil {
			written = append(written, f)
		} else {
			included = append(included, f)
		}
	}

	var out []ast.Member
	for len(included) > 0 {
		name, n := u.include(included)
		out = append(out, &ast.Include{Type: &ast.ClassRef{N: name}})
		included = included[n:]
	}
	for _, f := range written {
		out = append(out, u.field(f))
	}
	return out
}

// include finds the mixin whose fields begin a run of copied fields,
// returning its name and how many fields it accounts for.
//
// A field copied through a mixin that includes another records the inner
// mixin, so the run is matched against each mixin's whole field list, and
// the longest match wins.
func (u *unlowerer) include(run []*ir.Field) (string, int) {
	best, size := "", 0
	for _, d := range u.model.GetDecls() {
		s := d.GetStructure()
		if s.GetKind() != ir.StructKind_STRUCT_KIND_MIXIN || len(s.GetFields()) <= size || len(s.GetFields()) > len(run) {
			continue
		}
		name := d.GetMeta().GetName()
		if matches(run, s.GetFields(), name) {
			best, size = name, len(s.GetFields())
		}
	}
	if size > 0 {
		return best, size
	}

	// No mixin accounts for the run; name the one the first field records,
	// and take every field recording it.
	from := run[0].GetIncludedFrom().GetName()
	n := 1
	for n < len(run) && run[n].GetIncludedFrom().GetName() == from {
		n++
	}
	return from, n
}

// matches reports whether run begins with a copy of a mixin's fields.
func matches(run, mixin []*ir.Field, name string) bool {
	for i, f := range mixin {
		from := name
		if f.GetIncludedFrom() != nil {
			from = f.GetIncludedFrom().GetName()
		}
		if run[i].GetMeta().GetName() != f.GetMeta().GetName() || run[i].GetIncludedFrom().GetName() != from {
			return false
		}
	}
	return true
}

func (u *unlowerer) field(f *ir.Field) *ast.Field {
	return &ast.Field{
		DeclHead:    declHead(f.GetMeta()),
		Owned:       f.GetOwned(),
		Type:        u.typeRef(f.GetType()),
		Constraints: u.constraints(f.GetConstraints()),
		Default:     literal(f.GetDefaultValue()),
	}
}

// instance rebuilds an instance in the `instance C<T>` form lowering
// rewrites `instance C for T` to.
func (u *unlowerer) instance(inst *ir.Instance) *ast.InstanceDecl {
	out := &ast.InstanceDecl{
		DeclHead: declHead(inst.GetMeta()),
		Params:   params(inst.GetParams()),
		Class:    u.classRef(inst.GetClass()),
		Requires: u.classRefs(inst.GetRequires()),
	}
	for _, b := range inst.GetBinds() {
		out.Binds = append(out.Binds, &ast.AssocTypeBind{N: b.GetName(), Target: u.typeRef(b.GetType())})
	}
	return out
}

// typeRef rebuilds a type reference, with sugar where the type records it
// was written that way and the form can carry it.
func (u *unlowerer) typeRef(id *ir.ID) *ast.TypeRef {
	t := u.model.Type(id)
	if t == nil {
		return &ast.TypeRef{N: id.GetName()}
	}

	args := t.GetArgs()
	switch t.GetWrote() {
	case ir.SyntacticForm_SYNTACTIC_FORM_BRACKETS:
		if len(args) == 1 {
			return &ast.TypeRef{List: u.typeRef(args[0])}
		}
	case ir.SyntacticForm_SYNTACTIC_FORM_BRACES:
		if len(args) == 1 {
			return &ast.TypeRef{Set: u.typeRef(args[0])}
		}
	case ir.SyntacticForm_SYNTACTIC_FORM_ARROW:
		if len(args) == 2 {
			return &ast.TypeRef{MapKey: u.typeRef(args[0]), MapValue: u.typeRef(args[1])}
		}
	case ir.SyntacticForm_SYNTACTIC_FORM_QUESTION:
		// `T?` follows `T` and precedes `| null`.
		if len(args) == 1 {
			if inner := u.typeRef(args[0]); !inner.Optional && !inner.Nullable {
				inner.Optional = true
				return inner
			}
		}
	case ir.SyntacticForm_SYNTACTIC_FORM_OR_NULL:
		if len(args) == 1 {
			if inner := u.typeRef(args[0]); !inner.Nullable {
				inner.Nullable = true
				return inner
			}
		}
	}

	out := &ast.TypeRef{Args: u.typeArgs(args)}
	switch {
	case t.GetParam() != nil:
		out.N = t.GetParam().GetName()
	case t.GetExtern() != nil:
		out.Qualifier, out.N = u.externName(t.GetExtern())
	default:
		out.N = t.GetCtor().GetName()
	}
	return out
}

func (u *unlowerer) typeArgs(ids []*ir.ID) []*ast.TypeArg {
	var out []*ast.TypeArg
	for _, id := range ids {
		if t := u.model.Type(id); t.GetUnit() != nil {
			out = append(out, u.unitArg(t.GetUnit()))
			continue
		}
		out = append(out, &ast.TypeArg{Type: u.typeRef(id)})
	}
	return out
}

// externName is how this file names a declaration in another package: by
// its import alias, or bare when a `_` import merged it in.
func (u *unlowerer) externName(id *ir.ID) (qualifier, name string) {
	idx := id.GetIndex()
	if idx < 0 || int(idx) >= len(u.model.GetExterns()) {
		// An unresolved `alias.Name` keeps what was written.
		if q, n, ok := strings.Cut(id.GetName(), "."); ok {
			return q, n
		}
		return "", id.GetName()
	}
	ext := u.model.GetExterns()[idx]
	return u.aliases[ext.GetPackage()], ext.GetName()
}

func (u *unlowerer) classRefs(refs []*ir.ClassRef) []*ast.ClassRef {
	var out []*ast.ClassRef
	for _, r := range refs {
		out = append(out, u.classRef(r))
	}
	return out
}

func (u *unlowerer) classRef(r *ir.ClassRef) *ast.ClassRef {
	out := &ast.ClassRef{Args: u.typeArgs(r.GetArgs())}
	if r.GetExtern() != nil {
		out.Qualifier, out.N = u.externName(r.GetExtern())
	} else {
		out.N = r.GetClass().GetName()
	}
	return out
}

// unitArg rebuilds a unit written as a type argument: by the name of a
// unit declaration measuring it, or else in the spelling that first named
// it.
func (u *unlowerer) unitArg(id *ir.ID) *ast.TypeArg {
	for _, d := range u.model.GetDecls() {
		if def := d.GetUnit(); def != nil && own(d) && def.GetUnit().GetIndex() == id.GetIndex() {
			return &ast.TypeArg{Type: &ast.TypeRef{N: d.GetMeta().GetName()}}
		}
	}
	if decl, ok := parseUnit("type T: decimal<" + u.unitWrote(id) + ">"); ok {
		if n, ok := decl.(*ast.NewtypeDecl); ok && len(n.Base.Args) == 1 {
			return n.Base.Args[0]
		}
	}
	return &ast.TypeArg{Unit: u.dimsExpr(id)}
}

// unitExpr rebuilds the expression a derived unit declaration names.
func (u *unlowerer) unitExpr(id *ir.ID) *ast.UnitExpr {
	if decl, ok := parseUnit("unit U = " + u.unitWrote(id)); ok {
		if d, ok := decl.(*ast.UnitDecl); ok && d.Expr != nil {
			return d.Expr
		}
	}
	return u.dimsExpr(id)
}

func (u *unlowerer) unit(id *ir.ID) *ir.Unit {
	idx := id.GetIndex()
	if idx < 0 || int(idx) >= len(u.model.GetUnits()) {
		return nil
	}
	return u.model.GetUnits()[idx]
}

func (u *unlowerer) unitWrote(id *ir.ID) string {
	if unit := u.unit(id); unit.GetWrote() != "" {
		return unit.GetWrote()
	}
	return id.GetName()
}

// dimsExpr writes a quantity as a product of its base units, for a unit
// whose spelling does not parse.
func (u *unlowerer) dimsExpr(id *ir.ID) *ast.UnitExpr {
	e := &ast.UnitExpr{}
	for _, d := range u.unit(id).GetDims() {
		op := "*"
		if len(e.Terms) == 0 {
			op = ""
		}
		e.Terms = append(e.Terms, &ast.UnitTerm{Op: op, N: d.GetBase().GetName(), Exp: int(d.GetExponent())})
	}
	return e
}

// parseUnit parses a one-declaration file, which is how a unit's recorded
// spelling becomes an expression again.
func parseUnit(src string) (ast.Decl, bool) {
	f, err := parser.Parse("unit", strings.NewReader(src))
	if err != nil || len(f.Decls) != 1 {
		return nil, false
	}
	return f.Decls[0], true
}

func (u *unlowerer) constraints(cs []*ir.Constraint) []*ast.Constraint {
	var out []*ast.Constraint
	for _, c := range cs {
		out = append(out, &ast.Constraint{N: c.GetName(), Args: literals(c.GetArgs())})
	}
	return out
}

func literals(in []*ir.Literal) []*ast.Literal {
	var out []*ast.Literal
	for _, l := range in {
		out = append(out, literal(l))
	}
	return out
}

func literal(l *ir.Literal) *ast.Literal {
	if l == nil {
		return nil
	}

	out := &ast.Literal{Text: l.GetText()}
	switch l.GetKind() {
	case ir.LiteralKind_LITERAL_KIND_STRING:
		out.Kind = ast.LitString
	case ir.LiteralKind_LITERAL_KIND_INT:
		out.Kind = ast.LitInt
	case ir.LiteralKind_LITERAL_KIND_FLOAT:
		out.Kind = ast.LitFloat
	case ir.LiteralKind_LITERAL_KIND_BOOL:
		out.Kind = ast.LitBool
	case ir.LiteralKind_LITERAL_KIND_NAME:
		out.Kind = ast.LitName
	case ir.LiteralKind_LITERAL_KIND_REGEX:
		out.Kind = ast.LitRegex
	case ir.LiteralKind_LITERAL_KIND_LIST:
		out.Kind = ast.LitList
		out.Items = literals(l.GetItems())
	case ir.LiteralKind_LITERAL_KIND_RANGE:
		out.Kind = ast.LitRange
		out.Text = ""
		if r := l.GetRange(); r != nil {
			out.Lo = bound(r.Low)
			out.Hi = bound(r.High)
		}
	}
	return out
}

func bound(n *int64) *ast.Literal {
	if n == nil {
		return nil
	}
	return &ast.Literal{Kind: ast.LitInt, Text: strconv.FormatInt(*n, 10)}
}

func declHead(m *ir.Meta) ast.DeclHead {
	h := ast.DeclHead{N: m.GetName(), Doc: m.GetDoc(), DocP: docPositions(m.GetDoc())}
	if dep := m.GetDeprecated(); dep != nil {
		h.Dep = &ast.Deprecation{Reason: dep.GetReason()}
	}
	return h
}

// docPositions gives each doc line a position, so the printer, which
// interleaves doc lines with comments by position, has one to read.
func docPositions(doc []string) []ast.Position {
	if len(doc) == 0 {
		return nil
	}
	return make([]ast.Position, len(doc))
}

func params(in []*ir.Param) []*ast.TypeParam {
	var out []*ast.TypeParam
	for _, p := range in {
		out = append(out, &ast.TypeParam{N: p.GetName(), Kind: kind(p.GetKind())})
	}
	return out
}

func kind(k *ir.Kind) *ast.Kind {
	if k == nil {
		return nil
	}
	out := &ast.Kind{Arrow: kind(k.GetArrow())}
	switch {
	case k.GetParen() != nil:
		out.Paren = kind(k.GetParen())
	case k.GetAtom() == ir.KindAtom_KIND_ATOM_UNIT:
		out.N = "unit"
	default:
		out.N = "type"
	}
	return out
}
