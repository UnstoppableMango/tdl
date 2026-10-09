// Package thrift generates a Thrift IDL file from a resolved model. The
// mapping is in docs/design/schema-backends.md.
package thrift

import (
	"context"
	"fmt"
	"math"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/unstoppablemango/tdl/backend/internal/emit"
	"github.com/unstoppablemango/tdl/internal/unlower"
	"github.com/unstoppablemango/tdl/ir"
	"github.com/unstoppablemango/tdl/plugin"
)

// Name names this backend in a target block and as tdl-gen-thrift.
const Name = "thrift"

// Backend implements [plugin.Backend].
type Backend struct{}

func (Backend) Describe() plugin.Description {
	str := []ir.LiteralKind{ir.LiteralKind_LITERAL_KIND_STRING}
	return plugin.Description{
		Name:    Name,
		Version: "0.1.0",
		// Each request stands alone.
		Reuse: true,
		Directives: []*plugin.DirectiveSpec{
			// The dotted namespace, written as `namespace *`; defaults to
			// the model's package.
			{Name: "package", MinArgs: 1, MaxArgs: 1, ArgKinds: str},
			// The Thrift name for a declaration, field, enum value, or
			// variant struct.
			{Name: "name", MinArgs: 1, MaxArgs: 1, ArgKinds: str},
			// The id of a field or variant, or an enum value.
			{Name: "number", MinArgs: 1, MaxArgs: 1, ArgKinds: []ir.LiteralKind{ir.LiteralKind_LITERAL_KIND_INT}},
			// Writes, as tdl.* annotations, every fact the IDL alone would
			// lose, so import rebuilds the model.
			{Name: "roundtrip"},
		},
		Reverse: true,
	}
}

var (
	// A Thrift field id is an i16.
	fieldNumbers = emit.NumberRule{Max: math.MaxInt16}
	enumNumbers  = emit.NumberRule{Max: math.MaxInt32}
)

// scalars maps a prelude primitive to its Thrift type. Thrift has no
// decimal, UUID, time, or unsigned type: the first three are strings, uint32
// widens to i64, and uint64 is unmapped.
var scalars = map[string]string{
	"string":   "string",
	"int":      "i64",
	"int32":    "i32",
	"int64":    "i64",
	"uint32":   "i64",
	"float32":  "double",
	"float64":  "double",
	"bool":     "bool",
	"bytes":    "binary",
	"decimal":  "string",
	"uuid":     "string",
	"instant":  "string",
	"date":     "string",
	"duration": "string",
}

var keywords = map[string]bool{}

func init() {
	for _, k := range strings.Fields(`
		binary bool byte const cpp_include cpp_type double enum exception extends
		false i16 i32 i64 i8 include list map namespace oneway optional required
		senum service set slist string struct throws true typedef union void`) {
		keywords[k] = true
	}
}

var ident = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

type generator struct {
	*emit.Session
	ns string
}

// rendered is one declaration's text, held until the cascade and the
// order decide whether and where it is emitted. text stops before the
// declaration's own annotations.
type rendered struct {
	text  string
	names []string
	meta  *ir.Meta
	kvs   []kv
}

// Generate returns one .thrift file holding the model's declarations.
func (Backend) Generate(_ context.Context, req *plugin.Request) (*plugin.Response, error) {
	g := &generator{Session: emit.NewSession(req, "Thrift")}

	g.ns = req.GetModel().GetPackage()
	var nsPos *ir.Position
	if d, ok := g.Block("package"); ok {
		g.ns, nsPos = d.GetArgs()[0].GetText(), d.GetPosition()
	}
	// An invalid namespace fails the whole output.
	if !validNamespace(g.ns) {
		g.Error(nsPos, "%q is not a Thrift namespace", g.ns)
		return g.Response(nil), nil
	}

	g.Roundtrip = g.Bare("roundtrip")
	if pkg := req.GetModel().GetPackage(); g.ns != pkg {
		g.Lose(emit.LossName, nsPos, "package %s is written %s, which reads back as the package", pkg, g.ns)
	}
	if len(req.GetModel().GetDoc()) > 0 {
		g.Lose(emit.LossDoc, nil, "Thrift has no place for the package's doc comment")
	}

	own := g.Own()
	out := map[*ir.Decl]*rendered{}
	skipped := map[*ir.Decl]bool{}
	// Each declaration's warnings are held until the cascade decides
	// whether it is emitted, since a skipped one loses everything at once.
	held := map[*ir.Decl][]*plugin.Diagnostic{}
	for _, d := range own {
		mark := len(g.Diags)
		r, err := g.decl(d)
		if err != nil {
			g.Diags = g.Diags[:mark]
			g.Skip(err)
			skipped[d] = true
			continue
		}
		held[d] = slices.Clone(g.Diags[mark:])
		g.Diags = g.Diags[:mark]
		if r != nil {
			out[d] = r
		}
	}

	// Only a surviving declaration claims names, and a name clash skips a
	// declaration, which can cascade, so both run to a fixed point.
	mark := len(g.Diags)
	for {
		g.Cascade(own, skipped)
		if !g.claim(own, out, skipped) {
			break
		}
	}
	if g.Roundtrip {
		// A skipped declaration is carried as TDL, so nothing is lost.
		g.Diags = g.Diags[:mark]
	}
	for _, d := range own {
		if !skipped[d] {
			g.Diags = append(g.Diags, held[d]...)
		}
	}

	emitted := func(d *ir.Decl) bool { return out[d] != nil && !skipped[d] }
	order := g.order(own, emitted)
	inPlace := slices.Equal(order, slices.DeleteFunc(slices.Clone(own), func(d *ir.Decl) bool { return !emitted(d) }))
	if !inPlace {
		g.Lose(emit.LossOrder, nil, "the declarations are written after what they name, which does not keep their order")
	}

	names := map[string]bool{}
	for _, d := range order {
		names[d.GetMeta().GetName()] = true
	}
	header := g.fileAnnotation(names)
	if len(order) == 0 && header == "" {
		return g.Response(nil), nil
	}

	var b strings.Builder
	b.WriteString("// Code generated by tdl. DO NOT EDIT.\n")
	switch {
	case g.ns != "":
		fmt.Fprintf(&b, "\nnamespace * %s%s\n", g.ns, header)
	case header != "":
		// The annotations need a namespace to hang on, and one for a
		// language no generator reads changes nothing.
		fmt.Fprintf(&b, "\nnamespace %s %s%s\n", carrier, carrier, header)
	}
	for _, d := range order {
		r := out[d]
		kvs := r.kvs
		if !inPlace && g.Roundtrip {
			kvs = append(kvs, kv{"tdl.at", strconv.Itoa(g.Index(d.GetMeta().GetName()))})
		}
		b.WriteString("\n")
		b.WriteString(r.text)
		b.WriteString(g.annotations(r.meta, kvs...))
		b.WriteString("\n")
	}

	path := "model.thrift"
	if g.ns != "" {
		path = emit.LastSegment(g.ns) + ".thrift"
	}
	return g.Response([]*plugin.File{{Path: path, Content: []byte(b.String())}}), nil
}

// order puts each declaration after everything it names; a cycle keeps
// declaration order.
func (g *generator) order(own []*ir.Decl, emitted func(*ir.Decl) bool) []*ir.Decl {
	var out []*ir.Decl
	visited := map[*ir.Decl]bool{}
	var visit func(d *ir.Decl)
	visit = func(d *ir.Decl) {
		if visited[d] {
			return
		}
		visited[d] = true
		for _, ref := range g.References(d) {
			if emitted(ref) {
				visit(ref)
			}
		}
		out = append(out, d)
	}
	for _, d := range own {
		if emitted(d) {
			visit(d)
		}
	}
	return out
}

// decl renders one declaration and the names it declares at file scope, or
// nil for one that declares nothing. Name clashes are [generator.claim]'s.
func (g *generator) decl(d *ir.Decl) (*rendered, error) {
	pos := d.GetMeta().GetPosition()
	name := d.GetMeta().GetName()

	switch {
	case d.GetClass() != nil:
		return nil, emit.Lost(emit.LossClass, pos, "%s is a class, and classes are not generated yet", name)
	case d.GetUnit() != nil:
		return nil, emit.Lost(emit.LossUnit, pos, "%s is a unit, and units are not generated yet", name)
	case d.GetAlias() != nil:
		if len(d.Params()) == 0 {
			g.Lose(emit.LossAlias, pos, "alias %s is expanded where used", name)
			g.typeLosses(d.GetAlias().GetTarget(), nil)
		}
		return nil, nil
	case d.GetStructure() == nil && d.GetEnumeration() == nil && d.GetNewtype() == nil:
		// A primitive declares nothing.
		return nil, nil
	}
	if len(d.Params()) > 0 {
		return nil, emit.Lost(emit.LossGeneric, pos, "%s is parameterized, and generics are not generated yet", name)
	}

	r := &rendered{meta: d.GetMeta()}
	var b strings.Builder
	var err error
	switch e := d.GetEnumeration(); {
	case d.GetNewtype() != nil:
		r.names, r.kvs, err = g.typedef(&b, d)
	case d.GetStructure() != nil:
		r.names, err = g.structure(&b, "struct", g.DeclName(d, emit.Pascal), d.GetMeta(), d.Fields(), true)
		r.kvs = g.structKVs(d, r.names)
	case emit.Fielded(e):
		r.names, r.kvs, err = g.union(&b, d)
	default:
		r.names, r.kvs, err = g.enum(&b, d)
	}
	if err != nil {
		return nil, err
	}

	for _, n := range r.names {
		if !ident.MatchString(n) {
			return nil, emit.Unsupported(pos, "%s would be named %s in Thrift, which is not an identifier", name, n)
		}
		if keywords[n] {
			return nil, emit.Unsupported(pos, "%s would be named %s in Thrift, which is a keyword", name, n)
		}
	}
	if !g.Roundtrip {
		g.WarnConstraints(d)
	}
	r.kvs = append(r.kvs, g.docKVs(d.GetMeta())...)
	r.text = b.String()
	return r, nil
}

// claim gives each surviving declaration its file-scope names, skipping one
// whose name an earlier one took, and reports whether it skipped any.
func (g *generator) claim(own []*ir.Decl, out map[*ir.Decl]*rendered, skipped map[*ir.Decl]bool) bool {
	names, changed := map[string]string{}, false
	for _, d := range own {
		if skipped[d] || out[d] == nil {
			continue
		}
		name, taken := d.GetMeta().GetName(), false
		for _, n := range out[d].names {
			other, ok := names[n]
			if !ok {
				continue
			}
			g.Skip(emit.Unsupported(d.GetMeta().GetPosition(),
				"%s would declare %s in Thrift, and %s already does", name, n, other))
			skipped[d], changed, taken = true, true, true
			break
		}
		if taken {
			continue
		}
		for _, n := range out[d].names {
			names[n] = name
		}
	}
	return changed
}

func (g *generator) typedef(b *strings.Builder, d *ir.Decl) ([]string, []kv, error) {
	ref, err := g.Resolve(d.GetNewtype().GetBase())
	if err != nil {
		return nil, nil, err
	}
	base, err := g.typ(ref)
	if err != nil {
		return nil, nil, err
	}
	name := g.DeclName(d, emit.Pascal)
	kvs := g.nameKVs(d, name)
	g.typeLosses(d.GetNewtype().GetBase(), nil)
	if src := g.newtypeSource(d); src != d.GetMeta().GetName()+": "+g.spell(ref) {
		kvs = append(kvs, kv{"tdl.source", src})
	}
	comment(b, "", d.GetMeta())
	fmt.Fprintf(b, "typedef %s %s", base, name)
	return []string{name}, kvs, nil
}

// structure renders a struct body, through its closing brace. top is
// whether it is a declaration's own, rather than a variant's.
func (g *generator) structure(b *strings.Builder, keyword, name string, meta *ir.Meta, fields []*ir.Field, top bool) ([]string, error) {
	nums, err := g.Numbers(name, emit.FieldMembers(fields), fieldNumbers)
	if err != nil {
		return nil, err
	}
	g.CheckPins(name, emit.FieldMembers(fields), nums, fieldNumbers)
	includes := make([]string, len(fields))
	if top {
		includes = unlower.Includes(g.Model, fields)
	}
	for i, inc := range includes {
		if inc != "" && (i == 0 || includes[i-1] != inc) {
			g.Lose(emit.LossInclude, fields[i].GetMeta().GetPosition(), "%s's include of %s is flattened into its fields", name, inc)
		}
	}

	var body strings.Builder
	seen := map[string]bool{}
	for i, f := range fields {
		ref, err := g.Resolve(f.GetType())
		if err != nil {
			return nil, err
		}
		optional, typ, err := g.fieldType(ref)
		if err != nil {
			return nil, err
		}

		field := g.FieldName(f, asWritten)
		if err := g.member(seen, name, field, f.GetMeta().GetPosition()); err != nil {
			return nil, err
		}

		// A non-optional field keeps default requiredness: Thrift advises
		// against `required`, which can never be relaxed.
		label := ""
		if optional {
			label = "optional "
		}
		kvs := g.fieldKVs(name, f, ref, field, includes[i])
		comment(&body, "  ", f.GetMeta())
		fmt.Fprintf(&body, "  %d: %s%s %s%s\n", nums[i], label, typ, field, g.annotations(f.GetMeta(), kvs...))
	}

	comment(b, "", meta)
	fmt.Fprintf(b, "%s %s {\n%s}", keyword, name, body.String())
	return []string{name}, nil
}

// enum renders an enum whose variants carry no fields. Thrift scopes the
// values to the enum, so they carry no prefix.
func (g *generator) enum(b *strings.Builder, d *ir.Decl) ([]string, []kv, error) {
	name := g.DeclName(d, emit.Pascal)
	variants := d.GetEnumeration().GetVariants()
	nums, err := g.Numbers(name, emit.VariantMembers(variants), enumNumbers)
	if err != nil {
		return nil, nil, err
	}
	g.CheckPins(name, emit.VariantMembers(variants), nums, enumNumbers)

	var body strings.Builder
	seen := map[string]bool{}
	for i, v := range variants {
		value := g.VariantName(v, emit.ScreamingSnake)
		if err := g.member(seen, name, value, v.GetMeta().GetPosition()); err != nil {
			return nil, nil, err
		}
		var kvs []kv
		if back := emit.FromScreaming(value); back != v.GetMeta().GetName() {
			g.Lose(emit.LossName, v.GetMeta().GetPosition(), "%s.%s is written %s, which reads back as %s", name, v.GetMeta().GetName(), value, back)
			kvs = append(kvs, kv{"tdl.name", v.GetMeta().GetName()})
		} else {
			g.CheckRenamed(v.GetDirectives(), emit.ScreamingSnake(back) != value, "%s.%s", name, v.GetMeta().GetName())
		}
		kvs = append(kvs, g.docKVs(v.GetMeta())...)
		comment(&body, "  ", v.GetMeta())
		fmt.Fprintf(&body, "  %s = %d%s\n", value, nums[i], g.annotations(v.GetMeta(), kvs...))
	}

	comment(b, "", d.GetMeta())
	fmt.Fprintf(b, "enum %s {\n%s}", name, body.String())
	return []string{name}, append(g.nameKVs(d, name), g.conformsKV(d)...), nil
}

// union renders an enum where any variant carries fields: a union of one
// struct per variant, empty for a fieldless one.
func (g *generator) union(b *strings.Builder, d *ir.Decl) ([]string, []kv, error) {
	name := g.DeclName(d, emit.Pascal)
	variants := d.GetEnumeration().GetVariants()
	nums, err := g.Numbers(name, emit.VariantMembers(variants), fieldNumbers)
	if err != nil {
		return nil, nil, err
	}
	g.CheckPins(name, emit.VariantMembers(variants), nums, fieldNumbers)

	declared := []string{name}
	var members strings.Builder
	seen := map[string]bool{}
	for i, v := range variants {
		pos := v.GetMeta().GetPosition()
		vname := v.GetMeta().GetName()
		structName := g.VariantName(v, func(s string) string { return name + emit.Pascal(s) })
		if slices.Contains(declared, structName) {
			return nil, nil, emit.Unsupported(pos, "%s.%s would declare %s in Thrift, which %s already declares", name, vname, structName, name)
		}
		if _, err := g.structure(b, "struct", structName, v.GetMeta(), v.GetFields(), false); err != nil {
			return nil, nil, err
		}
		b.WriteString(g.annotations(v.GetMeta(), g.docKVs(v.GetMeta())...))
		b.WriteString("\n\n")
		declared = append(declared, structName)

		field := emit.Camel(vname)
		if err := g.member(seen, name, field, pos); err != nil {
			return nil, nil, err
		}
		var kvs []kv
		back := variantBack(name, structName, field)
		if back != vname {
			g.Lose(emit.LossName, pos, "%s.%s is written %s, which reads back as %s", name, vname, field, back)
			kvs = append(kvs, kv{"tdl.name", vname})
		} else {
			g.CheckRenamed(v.GetDirectives(), structName != name+back, "%s.%s", name, vname)
		}
		fmt.Fprintf(&members, "  %d: %s %s%s\n", nums[i], structName, field, g.annotations(v.GetMeta(), kvs...))
	}

	comment(b, "", d.GetMeta())
	fmt.Fprintf(b, "union %s {\n%s}", name, members.String())
	return declared, append(g.nameKVs(d, name), g.conformsKV(d)...), nil
}

// member claims a name inside a struct, a union, or an enum.
func (g *generator) member(seen map[string]bool, owner, name string, pos *ir.Position) error {
	if !ident.MatchString(name) {
		return emit.Unsupported(pos, "%s.%s is not a Thrift identifier", owner, name)
	}
	if keywords[name] {
		return emit.Unsupported(pos, "%s.%s is a Thrift keyword", owner, name)
	}
	if seen[name] {
		return emit.Unsupported(pos, "%s has two members named %s in Thrift", owner, name)
	}
	seen[name] = true
	return nil
}

// fieldType returns whether a field is optional, and its type.
func (g *generator) fieldType(r *emit.Ref) (bool, string, error) {
	if r.Form != emit.Option && r.Form != emit.Nullable {
		typ, err := g.typ(r)
		return false, typ, err
	}
	if r.Elem.Form == emit.Option || r.Elem.Form == emit.Nullable {
		return false, "", emit.Unsupported(r.Pos, "an optional value holding an optional value has no Thrift form")
	}
	typ, err := g.typ(r.Elem)
	return true, typ, err
}

// typ is the Thrift type for anything but an optional, which only a field
// can be.
func (g *generator) typ(r *emit.Ref) (string, error) {
	switch r.Form {
	case emit.Prim:
		typ, ok := scalars[r.Name]
		if !ok {
			return "", emit.Unsupported(r.Pos, "primitive %s has no Thrift type", r.Name)
		}
		return typ, nil
	case emit.List, emit.Set:
		elem, err := g.typ(r.Elem)
		if err != nil {
			return "", err
		}
		if r.Form == emit.Set {
			return "set<" + elem + ">", nil
		}
		return "list<" + elem + ">", nil
	case emit.Map:
		key, err := g.typ(r.Key)
		if err != nil {
			return "", err
		}
		val, err := g.typ(r.Elem)
		if err != nil {
			return "", err
		}
		return "map<" + key + ", " + val + ">", nil
	case emit.Option, emit.Nullable:
		return "", emit.Unsupported(r.Pos, "a Thrift container holds no nulls, so an optional value inside one has no Thrift form")
	}
	return g.DeclName(r.Decl, emit.Pascal), nil
}

func asWritten(name string) string { return name }

// comment writes a node's documentation and deprecation as a doc comment.
func comment(b *strings.Builder, indent string, meta *ir.Meta) {
	lines := emit.Doc(meta)
	if reason, ok := emit.Deprecated(meta); ok && reason != "" {
		if len(lines) > 0 {
			lines = append(lines, "")
		}
		// A reason may span lines, each needing the comment prefix.
		lines = append(lines, strings.Split("Deprecated: "+reason, "\n")...)
	}
	emit.BlockComment(b, indent, lines)
}

// quote writes prose as a Thrift string literal. `\"` and `\\` are the only
// escapes Thrift reads, so a control character becomes a space.
func quote(s string) string {
	var b strings.Builder
	b.WriteByte('"')
	for _, r := range s {
		switch {
		case r == '"' || r == '\\':
			b.WriteByte('\\')
			b.WriteRune(r)
		case r < 0x20 || r == 0x7f:
			b.WriteByte(' ')
		default:
			b.WriteRune(r)
		}
	}
	b.WriteByte('"')
	return b.String()
}

func validNamespace(ns string) bool {
	if ns == "" {
		return true
	}
	for seg := range strings.SplitSeq(ns, ".") {
		if !ident.MatchString(seg) {
			return false
		}
	}
	return true
}
