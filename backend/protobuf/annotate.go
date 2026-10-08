package protobuf

import (
	"cmp"
	_ "embed"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"unicode"

	"github.com/unstoppablemango/tdl/ast"
	"github.com/unstoppablemango/tdl/backend/internal/emit"
	"github.com/unstoppablemango/tdl/internal/unlower"
	"github.com/unstoppablemango/tdl/ir"
)

// annotationsFile is where the output places tdl/annotations.proto, and the
// path a file using its options imports.
const annotationsFile = "tdl/annotations.proto"

// annotationsProto declares the options a `roundtrip` directive writes.
//
//go:embed tdl/annotations.proto
var annotationsProto []byte

// bare reports whether the target block carries an argument-less directive.
func (g *generator) bare(name string) bool {
	for _, block := range g.Model.GetTargets() {
		if block.GetMeta().GetName() == g.Target && g.tagged(block.GetDirectives(), name) {
			return true
		}
	}
	return false
}

// lose warns that a fact is lost, unless an annotation carries it.
func (g *generator) lose(code string, pos *ir.Position, format string, args ...any) {
	if !g.roundtrip {
		g.Lossy(code, pos, format, args...)
	}
}

// skip reports a declaration that is not generated. Under roundtrip it is
// carried as TDL instead, so nothing is reported.
func (g *generator) skip(err error) {
	if !g.roundtrip {
		g.Warn(err)
	}
}

// spansFiles reports whether the declarations are placed in more than one
// file, which loses their order across files.
func (g *generator) spansFiles(own []*ir.Decl) bool {
	paths := map[string]bool{}
	for _, d := range own {
		if p, err := g.pathOf(d); err == nil && d.GetAlias() == nil && d.GetNewtype() == nil && d.GetPrimitive() == nil {
			paths[p] = true
		}
	}
	if len(paths) > 1 {
		g.lose(emit.LossOrder, nil, "the declarations are placed in %d files, which do not keep their order", len(paths))
		return true
	}
	return false
}

// source is the model unlowered, built once.
func (g *generator) unlowered() *ast.File {
	if g.source == nil {
		g.source = unlower.File(g.Model)
	}
	return g.source
}

// kv is one field of an annotation's message literal.
type kv struct{ key, text string }

// text is a string-valued field, or nothing for "".
func text(key, value string) kv {
	if value == "" {
		return kv{}
	}
	return kv{key, quote(value)}
}

// annotation is the option setting one of tdl/annotations.proto's
// extensions, or "" when it would be empty or roundtrip is off.
func (g *generator) annotation(ext string, kvs ...kv) string {
	if !g.roundtrip {
		return ""
	}
	var parts []string
	for _, f := range kvs {
		if f.key != "" {
			parts = append(parts, f.key+": "+f.text)
		}
	}
	if len(parts) == 0 {
		return ""
	}
	g.annotated = true
	return "(tdl." + ext + ") = {" + strings.Join(parts, " ") + "}"
}

// annotate appends an annotation to a field's or enum value's options.
func (g *generator) annotate(opts []string, ext string, kvs ...kv) []string {
	if a := g.annotation(ext, kvs...); a != "" {
		opts = append(opts, a)
	}
	return opts
}

// checkRenamed warns when import would write a `name` directive and the
// model has none, or the reverse: written is what the node is named in
// protobuf, and readerWrites whether that differs from the convention.
func (g *generator) checkRenamed(dirs []*ir.Directive, readerWrites bool, format string, args ...any) {
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

// checkPins warns when import would pin a member's number differently: it
// pins the fewest members that give every member its number.
func (g *generator) checkPins(owner string, members []emit.Member, nums []int64, rule emit.NumberRule) {
	pins := emit.Pins(nums, rule)
	for i, m := range members {
		d, has := g.Find(m.Directives, "number")
		switch {
		case has && !pins[i]:
			g.lose(emit.LossNumber, d.GetPosition(), "%s.%s is pinned to %d, which it would be given anyway, and the pin does not read back", owner, m.Name, nums[i])
		case !has && pins[i]:
			g.lose(emit.LossNumber, m.Position, "%s.%s reads back pinned to %d", owner, m.Name, nums[i])
		}
	}
}

// declAnnotation is a message's or enum's annotation: its TDL name when
// the protobuf name does not read back as it, and its place in the file
// when the output spans files.
func (g *generator) declAnnotation(ext string, d *ir.Decl, written string, extra ...kv) string {
	name := d.GetMeta().GetName()
	var kvs []kv
	if written != name {
		g.lose(emit.LossName, d.GetMeta().GetPosition(), "%s is written %s, which reads back as its name", name, written)
		kvs = append(kvs, text("name", name))
	} else {
		g.checkRenamed(d.GetDirectives(), emit.Pascal(written) != written, "%s", name)
	}
	kvs = append(kvs, extra...)
	if g.multi {
		kvs = append(kvs, kv{"at", strconv.Itoa(g.index(name))})
	}
	return g.annotation(ext, kvs...)
}

// structAnnotation is [generator.declAnnotation] for a struct, with its
// kind, since every struct reads back as a value.
func (g *generator) structAnnotation(d *ir.Decl, written string) string {
	s := d.GetStructure()
	name := d.GetMeta().GetName()
	var kind kv
	switch s.GetKind() {
	case ir.StructKind_STRUCT_KIND_ENTITY:
		g.lose(emit.LossStructKind, d.GetMeta().GetPosition(), "%s is an entity, and reads back as a value", name)
		kind = kv{"kind", "KIND_ENTITY"}
	case ir.StructKind_STRUCT_KIND_MIXIN:
		g.lose(emit.LossStructKind, d.GetMeta().GetPosition(), "%s is a mixin, and reads back as a value", name)
		kind = kv{"kind", "KIND_MIXIN"}
	}
	var conforms kv
	for _, c := range s.GetConforms() {
		if n := c.GetClass().GetName(); n != "Entity" || c.GetExtern() != nil || len(c.GetArgs()) > 0 {
			g.lose(emit.LossClass, c.GetPosition(), "%s conforms to %s, which protobuf does not carry", name, n)
			conforms = text("conforms", g.conforms(name))
		}
	}
	return g.declAnnotation("message", d, written, kind, conforms)
}

// conforms is a struct's conformance list as TDL: `Entity, Auditable`.
func (g *generator) conforms(name string) string {
	for _, decl := range g.unlowered().Decls {
		if sd, ok := decl.(*ast.StructDecl); ok && sd.N == name {
			head := ast.PrintDecl(&ast.StructDecl{DeclHead: ast.DeclHead{N: "T"}, Keyword: "type", Conforms: sd.Conforms})
			head = strings.TrimPrefix(head, "type T: ")
			return strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(head), "{ }"))
		}
	}
	return ""
}

// serviceAnnotation carries a service as TDL, since import reads back no
// service yet.
func (g *generator) serviceAnnotation(d *ir.Decl) string {
	name := d.GetMeta().GetName()
	g.lose(emit.LossUnsupported, d.GetMeta().GetPosition(), "service %s does not read back yet", name)
	for _, decl := range g.unlowered().Decls {
		if isDecl(decl) && decl.Name() == name {
			return g.annotation("service", text("source", printItem(decl)))
		}
	}
	return ""
}

// index is a declaration's index among the unlowered file's.
func (g *generator) index(name string) int {
	for i, d := range g.unlowered().Decls {
		if isDecl(d) && d.Name() == name {
			return i
		}
	}
	return -1
}

// isDecl reports whether a top-level item is a declaration, which has a
// name of its own, rather than an instance or a target block.
func isDecl(d ast.Decl) bool {
	switch d.(type) {
	case *ast.InstanceDecl, *ast.TargetDecl:
		return false
	}
	return true
}

// fieldLosses warns about and annotates what a field loses: its name, how
// its type is written, and what protobuf has no form for.
func (g *generator) fieldLosses(opts []string, owner string, f *ir.Field, ref *emit.Ref, written, include string) []string {
	name := f.GetMeta().GetName()
	pos := f.GetMeta().GetPosition()
	back := emit.Camel(written)
	if back != name {
		g.lose(emit.LossName, pos, "%s.%s is written %s, which reads back as %s", owner, name, written, back)
	} else {
		g.checkRenamed(f.GetDirectives(), emit.Snake(back) != written, "%s.%s", owner, name)
	}
	if f.GetOwned() {
		g.lose(emit.LossOwned, pos, "%s.%s is owned, which protobuf does not carry", owner, name)
	}
	if f.GetDefaultValue() != nil {
		g.lose(emit.LossDefault, pos, "%s.%s has a default, which protobuf does not carry", owner, name)
	}
	g.typeLosses(f.GetType(), pos)

	var kvs []kv
	if src := g.fieldSource(f, false); src != back+": "+g.spell(ref) {
		kvs = append(kvs, text("source", src))
	}
	kvs = append(kvs, text("include", include))
	return g.annotate(opts, "field", kvs...)
}

// oneofLosses warns about what an inlined oneof loses, the enum it names
// above all, and returns its annotation.
func (g *generator) oneofLosses(owner string, f *ir.Field, e *ir.Decl, include string) string {
	name := f.GetMeta().GetName()
	pos := f.GetMeta().GetPosition()
	oneofName := emit.Snake(name)
	if back := emit.Camel(oneofName); back != name {
		g.lose(emit.LossName, pos, "%s.%s is written %s, which reads back as %s", owner, name, oneofName, back)
	}
	if back := emit.Pascal(oneofName); back != e.GetMeta().GetName() {
		g.lose(emit.LossName, pos, "%s.%s reads back naming its enum %s rather than %s", owner, name, back, e.GetMeta().GetName())
	}
	if len(e.GetMeta().GetDoc()) > 0 || e.GetMeta().GetDeprecated() != nil || f.GetMeta().GetDeprecated() != nil {
		g.lose(emit.LossDoc, pos, "%s.%s is a oneof, which carries neither its enum's doc comment nor a deprecation", owner, name)
	}
	if f.GetOwned() {
		g.lose(emit.LossOwned, pos, "%s.%s is owned, which protobuf does not carry", owner, name)
	}
	if f.GetDefaultValue() != nil {
		g.lose(emit.LossDefault, pos, "%s.%s has a default, which protobuf does not carry", owner, name)
	}
	for _, v := range e.GetEnumeration().GetVariants() {
		vf := v.GetFields()[0]
		member := emit.Snake(vf.GetMeta().GetName())
		if emit.Pascal(member) != v.GetMeta().GetName() || emit.Camel(member) != vf.GetMeta().GetName() {
			g.lose(emit.LossName, v.GetMeta().GetPosition(), "%s.%s's member %s reads back as variant %s", owner, name, member, emit.Pascal(member))
		}
		if len(v.GetMeta().GetDoc()) > 0 {
			g.lose(emit.LossDoc, v.GetMeta().GetPosition(), "%s.%s's doc comment joins its field's", e.GetMeta().GetName(), v.GetMeta().GetName())
		}
	}
	return g.annotation("oneof", text("source", g.fieldSource(f, true)), text("include", include))
}

// fieldSource is a field as TDL, without its doc comment, and without its
// deprecation unless deprecated is set.
func (g *generator) fieldSource(f *ir.Field, deprecated bool) string {
	af := unlower.Field(g.Model, f)
	if !deprecated {
		af.Dep = nil
	}
	return ast.PrintField(af)
}

// lossyPrimitives is the primitives whose protobuf type reads back as
// another.
var lossyPrimitives = map[string]bool{"int": true, "decimal": true, "uuid": true, "date": true}

// typeLosses warns about what a type loses in protobuf: a primitive's
// kind, a set's uniqueness, and sugar import writes another way. A newtype
// or alias it names warns where it is declared.
func (g *generator) typeLosses(id *ir.ID, at *ir.Position) {
	t := g.Model.Type(id)
	if t.GetUnit() != nil {
		g.lose(emit.LossUnit, cmp.Or(at, t.GetPosition()), "the unit %s is not carried", t.GetUnit().GetName())
		return
	}
	decl := g.Model.Decl(t.GetCtor())
	if decl == nil || decl.GetAlias() != nil || decl.GetNewtype() != nil {
		return
	}
	// A type is interned at its first use, so a field's own position says
	// which use is meant.
	pos := cmp.Or(at, t.GetPosition())
	name := decl.GetMeta().GetName()
	wrote := t.GetWrote()
	switch {
	case decl.GetPrimitive() != nil:
		switch {
		case name == "Set":
			g.lose(emit.LossCollection, pos, "a set is a repeated field, which reads back as a list")
		case name == "List" && wrote != ir.SyntacticForm_SYNTACTIC_FORM_BRACKETS,
			name == "Map" && wrote != ir.SyntacticForm_SYNTACTIC_FORM_ARROW:
			g.lose(emit.LossCollection, pos, "%s reads back written as sugar", name)
		case lossyPrimitives[name]:
			g.lose(emit.LossPrimitive, pos, "%s is protobuf %s, which reads back as %s", name, scalars[name], primitiveOf(scalars[name]))
		}
	case decl.GetEnumeration() != nil && name == "Nullable":
		g.lose(emit.LossOptional, pos, "a nullable value is optional in protobuf, and reads back as T?")
	case decl.GetEnumeration() != nil && name == "Option":
		if wrote != ir.SyntacticForm_SYNTACTIC_FORM_QUESTION {
			g.lose(emit.LossOptional, pos, "Option reads back written as T?")
		}
		if len(t.GetArgs()) == 1 {
			if ref, err := g.Resolve(t.GetArgs()[0]); err == nil {
				if inner, err := g.Expand(ref); err == nil && (g.edition != "" || isMessage(inner)) {
					g.lose(emit.LossOptional, pos, "every message field and every field under an edition has presence, so an optional one reads back as required")
				}
			}
		}
	}
	for _, arg := range t.GetArgs() {
		g.typeLosses(arg, at)
	}
}

// primitiveOf is the TDL primitive a protobuf scalar reads back as.
func primitiveOf(scalar string) string {
	switch scalar {
	case "float":
		return "float32"
	case "double":
		return "float64"
	case "google.protobuf.Timestamp":
		return "instant"
	case "google.protobuf.Duration":
		return "duration"
	}
	return scalar
}

// spell is the TDL a field of type r reads back as, without annotations.
func (g *generator) spell(r *emit.Ref) string {
	r, err := g.Expand(r)
	if err != nil {
		return ""
	}
	switch r.Form {
	case emit.Option, emit.Nullable:
		inner, err := g.Expand(r.Elem)
		if err != nil {
			return ""
		}
		if g.edition != "" || isMessage(inner) {
			return g.spell(inner)
		}
		return g.spell(inner) + "?"
	case emit.List, emit.Set:
		return "[" + g.spell(r.Elem) + "]"
	case emit.Map:
		return "{" + g.spell(r.Key) + " -> " + g.spell(r.Elem) + "}"
	case emit.Prim:
		return primitiveOf(scalars[r.Name])
	case emit.Extern:
		if f, ok := g.Find(r.Extern.GetDirectives(), "foreign"); ok {
			return emit.LastSegment(f.GetArgs()[1].GetText())
		}
		return emit.LastSegment(r.Extern.GetName())
	}
	if f, ok := g.Find(r.Decl.GetDirectives(), "foreign"); ok {
		return emit.LastSegment(f.GetArgs()[1].GetText())
	}
	return r.Decl.GetMeta().GetName()
}

// fileAnnotation is the (tdl.file) option: the TDL package when the
// protobuf one renames it, the imports, and every top-level item no message,
// enum, or service declares, by its index in the file.
func (g *generator) fileAnnotation(placed map[*ir.Decl]string) string {
	emitted := map[string]bool{}
	for d := range placed {
		emitted[d.GetMeta().GetName()] = true
	}

	var kvs []kv
	if pkg := g.Model.GetPackage(); pkg != g.pkg {
		kvs = append(kvs, text("package", pkg))
	}
	file := g.unlowered()
	for _, imp := range file.Imports {
		kvs = append(kvs, text("imports", strings.TrimSuffix(ast.Fprint(&ast.File{Imports: []*ast.ImportDecl{imp}}), "\n")))
	}
	for i, d := range file.Decls {
		if isDecl(d) && emitted[d.Name()] {
			continue
		}
		if t, ok := d.(*ast.TargetDecl); ok && t.N == g.Target {
			if d = withoutRoundtrip(t); d == nil {
				continue
			}
		}
		kvs = append(kvs, kv{"items", fmt.Sprintf("{at: %d source: %s}", i, quote(printItem(d)))})
	}
	if !g.roundtrip || len(kvs) == 0 {
		return ""
	}
	parts := make([]string, len(kvs))
	for i, f := range kvs {
		parts[i] = "  " + f.key + ": " + f.text
	}
	return "(tdl.file) = {\n" + strings.Join(parts, "\n") + "\n}"
}

// withoutRoundtrip is this target's block without the roundtrip
// directive, or nil when nothing else is in it.
func withoutRoundtrip(t *ast.TargetDecl) ast.Decl {
	out := *t
	out.Entries = slices.DeleteFunc(slices.Clone(t.Entries), func(e *ast.TargetEntry) bool {
		return e.Path == "" && e.Directive != nil && e.Directive.N == "roundtrip" && len(e.Directive.Args) == 0
	})
	if len(out.Entries) == 0 && len(t.Entries) > 0 {
		return nil
	}
	return &out
}

// printItem is a top-level item as TDL, doc comment included.
func printItem(d ast.Decl) string {
	return strings.TrimSuffix(ast.Fprint(&ast.File{Decls: []ast.Decl{d}}), "\n")
}

// valueBack is the variant name an enum value reads back as, and whether
// import writes a name directive to keep the value's name.
func valueBack(prefix, value string) (string, bool) {
	rest, ok := strings.CutPrefix(value, prefix+"_")
	if !ok || rest == "" {
		rest = value
	}
	back := fromScreaming(rest)
	return back, prefix+"_"+emit.ScreamingSnake(back) != value
}

// fromScreaming turns SCREAMING_SNAKE into Pascal case: IN_TRANSIT is
// InTransit.
func fromScreaming(s string) string {
	var b strings.Builder
	for w := range strings.SplitSeq(s, "_") {
		r := []rune(strings.ToLower(w))
		if len(r) == 0 {
			continue
		}
		r[0] = unicode.ToUpper(r[0])
		b.WriteString(string(r))
	}
	return b.String()
}

// quote writes s as a protobuf string literal, escaping only what the
// language requires, so UTF-8 stays readable.
func quote(s string) string {
	var b strings.Builder
	b.WriteByte('"')
	for _, c := range []byte(s) {
		switch c {
		case '"':
			b.WriteString(`\"`)
		case '\\':
			b.WriteString(`\\`)
		case '\n':
			b.WriteString(`\n`)
		case '\t':
			b.WriteString(`\t`)
		case '\r':
			b.WriteString(`\r`)
		default:
			if c < 0x20 || c == 0x7f {
				fmt.Fprintf(&b, `\%03o`, c)
				continue
			}
			b.WriteByte(c)
		}
	}
	b.WriteByte('"')
	return b.String()
}
