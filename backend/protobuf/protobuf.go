// Package protobuf generates a proto3 schema from a resolved model, or one
// under the edition an `edition` directive names.
//
// docs/design/schema-backends.md has the mapping and the reasons for it.
// Two things are worth knowing before reading the output. An enum whose
// variants carry fields is a message holding a oneof of one nested message
// per variant, which is protobuf's sum type. And every field, variant, and
// enum value without a `number` directive takes the lowest number no pin
// holds, in declaration order, because the IR has no numbers and protobuf
// cannot go without them.
package protobuf

import (
	"context"
	"fmt"
	"maps"
	"math"
	"path"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/unstoppablemango/tdl/backend/internal/emit"
	"github.com/unstoppablemango/tdl/ir"
	"github.com/unstoppablemango/tdl/plugin"
)

// Name is what this backend is called, in a target block and as
// tdl-gen-protobuf on PATH.
const Name = "protobuf"

// Backend implements [plugin.Backend].
type Backend struct{}

func (Backend) Describe() plugin.Description {
	str := []ir.LiteralKind{ir.LiteralKind_LITERAL_KIND_STRING}
	return plugin.Description{
		Name:    Name,
		Version: "0.1.0",
		// Each request is answered from the request alone.
		Reuse: true,
		Directives: []*plugin.DirectiveSpec{
			// The dotted protobuf package. The model's package is the
			// default.
			{Name: "package", MinArgs: 1, MaxArgs: 1, ArgKinds: str},
			// The file name within the package's directories, in place of
			// the package's last segment with .proto.
			{Name: "file", MinArgs: 1, MaxArgs: 1, ArgKinds: str},
			// The protobuf name for a declaration, a field, an enum value, or
			// a variant's nested message.
			{Name: "name", MinArgs: 1, MaxArgs: 1, ArgKinds: str},
			// The wire number of a field, a variant, or an enum value.
			{Name: "number", MinArgs: 1, MaxArgs: 1, ArgKinds: []ir.LiteralKind{ir.LiteralKind_LITERAL_KIND_INT}},
			// The protobuf edition the file declares in place of proto3.
			{Name: "edition", MinArgs: 1, MaxArgs: 1, ArgKinds: str},
			// Field numbers or names a message reserves, one `reserved`
			// statement per directive. arg_kinds constrains by position, so
			// it cannot say "int or string" and is left unset.
			{Name: "reserved", MinArgs: 1, MaxArgs: -1, Repeatable: true},
			// Inlines a field's sum type into its message as a oneof.
			{Name: "oneof"},
			// A file the output imports, for the options it uses.
			{Name: "import", MinArgs: 1, MaxArgs: 1, ArgKinds: str, Repeatable: true},
			// An option, written `name = value` in the brackets of a field or
			// an enum value, or as an `option` statement in a message or enum.
			{Name: "option", MinArgs: 2, MaxArgs: 2, ArgKinds: []ir.LiteralKind{ir.LiteralKind_LITERAL_KIND_STRING, ir.LiteralKind_LITERAL_KIND_STRING}, Repeatable: true},
		},
	}
}

var (
	// fieldNumbers covers message fields and oneof members, which share a
	// message's number space.
	fieldNumbers = emit.NumberRule{Max: 536870911, Reserved: [][2]int64{{19000, 19999}}}
	// enumNumbers starts at one because zero is the UNSPECIFIED value
	// protobuf requires first.
	enumNumbers = emit.NumberRule{Max: math.MaxInt32}
)

// scalars maps a prelude primitive to the protobuf type standing for it.
var scalars = map[string]string{
	"string":   "string",
	"int":      "int64",
	"int32":    "int32",
	"uint32":   "uint32",
	"int64":    "int64",
	"uint64":   "uint64",
	"float32":  "float",
	"float64":  "double",
	"bool":     "bool",
	"bytes":    "bytes",
	"decimal":  "string",
	"uuid":     "string",
	"date":     "string",
	"instant":  "google.protobuf.Timestamp",
	"duration": "google.protobuf.Duration",
}

// wellKnown is the file each well-known type is imported from.
var wellKnown = map[string]string{
	"google.protobuf.Timestamp": "google/protobuf/timestamp.proto",
	"google.protobuf.Duration":  "google/protobuf/duration.proto",
}

// mapKeys is the protobuf types a map may be keyed by that this backend
// emits: protobuf allows integral and string keys, and nothing else.
var mapKeys = map[string]bool{
	"string": true, "bool": true,
	"int32": true, "int64": true, "uint32": true, "uint64": true,
}

// editions is every edition the `edition` directive accepts.
var editions = map[string]bool{"2023": true, "2024": true}

// oneof is the name of the oneof a sum type's message holds.
const oneof = "variant"

var ident = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

type generator struct {
	*emit.Session
	pkg string
	// file is what the file directive names, or "" when there is none.
	file string

	// edition is what the `edition` directive names, or "" for proto3.
	edition string

	// names is every name declared at package scope, with the declaration
	// that declared it. Protobuf gives an enum's values package scope too.
	names map[string]string

	// imports is what the declaration being rendered needs.
	imports map[string]bool
}

// rendered is one declaration's text, held back until the cascade has
// decided whether it is emitted.
type rendered struct {
	decl    *ir.Decl
	text    string
	imports map[string]bool
}

// Generate returns one .proto file holding every declaration the model owns.
func (Backend) Generate(_ context.Context, req *plugin.Request) (*plugin.Response, error) {
	g := &generator{
		Session: emit.NewSession(req, "protobuf"),
		pkg:     req.GetModel().GetPackage(),
		names:   map[string]string{},
	}

	var pkgPos *ir.Position
	if d, ok := g.Block("package"); ok {
		g.pkg, pkgPos = d.GetArgs()[0].GetText(), d.GetPosition()
	}
	if d, ok := g.Block("edition"); ok {
		if g.edition = d.GetArgs()[0].GetText(); !editions[g.edition] {
			g.Error(d.GetPosition(), "%q is not a protobuf edition", g.edition)
			return g.Response(nil), nil
		}
	}
	// Every declaration lives in the package, so a name protobuf refuses is
	// the whole output rather than one declaration to skip.
	if !validPackage(g.pkg) {
		g.Error(pkgPos, "%q is not a protobuf package name", g.pkg)
		return g.Response(nil), nil
	}
	if d, ok := g.Block("file"); ok {
		g.file = d.GetArgs()[0].GetText()
		if strings.ContainsAny(g.file, `/\`) || !strings.HasSuffix(g.file, ".proto") {
			g.Error(d.GetPosition(), "%q is not a protobuf file name", g.file)
			return g.Response(nil), nil
		}
	}

	own := g.Own()
	inlinedOnly := g.inlinedOnly()
	skipped := map[*ir.Decl]bool{}
	var out []rendered
	for _, d := range own {
		if inlinedOnly[d] {
			continue
		}
		g.imports = map[string]bool{}
		text, err := g.decl(d)
		if err != nil {
			g.Warn(err)
			skipped[d] = true
			continue
		}
		if text != "" {
			out = append(out, rendered{d, text, g.imports})
		}
	}
	g.Cascade(own, skipped)

	var blocks []string
	need := map[string]bool{}
	for _, r := range out {
		if skipped[r.decl] {
			continue
		}
		blocks = append(blocks, r.text)
		maps.Copy(need, r.imports)
	}
	for _, block := range req.GetModel().GetTargets() {
		if block.GetMeta().GetName() != g.Target {
			continue
		}
		for _, d := range plugin.Directives(g.Target, block.GetDirectives()) {
			if d.GetName() == "import" && len(d.GetArgs()) > 0 {
				need[d.GetArgs()[0].GetText()] = true
			}
		}
	}
	if len(blocks) == 0 {
		return g.Response(nil), nil
	}

	var b strings.Builder
	header := `syntax = "proto3";`
	if g.edition != "" {
		header = fmt.Sprintf("edition = %q;", g.edition)
	}
	fmt.Fprintf(&b, "// Code generated by tdl. DO NOT EDIT.\n\n%s\n", header)
	if g.pkg != "" {
		fmt.Fprintf(&b, "\npackage %s;\n", g.pkg)
	}
	if len(need) > 0 {
		b.WriteString("\n")
		for _, imp := range slices.Sorted(maps.Keys(need)) {
			fmt.Fprintf(&b, "import %q;\n", imp)
		}
	}
	for _, block := range blocks {
		b.WriteString("\n")
		b.WriteString(block)
	}

	return g.Response([]*plugin.File{{Path: g.filePath(), Content: []byte(b.String())}}), nil
}

// decl renders one declaration, or "" for one that declares nothing.
func (g *generator) decl(d *ir.Decl) (string, error) {
	pos := d.GetMeta().GetPosition()
	name := d.GetMeta().GetName()

	switch {
	case d.GetClass() != nil:
		return "", emit.Unsupported(pos, "%s is a class, and classes are not generated yet", name)
	case d.GetUnit() != nil:
		return "", emit.Unsupported(pos, "%s is a unit, and units are not generated yet", name)
	case d.GetStructure() == nil && d.GetEnumeration() == nil && d.GetNewtype() == nil:
		// An alias is expanded where it is used, and a model's own
		// primitive names an opaque root; neither declares anything.
		return "", nil
	}
	if len(d.Params()) > 0 {
		return "", emit.Unsupported(pos, "%s is parameterized, and generics are not generated yet", name)
	}

	if n := d.GetNewtype(); n != nil {
		// A newtype is expanded where it is used, since a wrapper message
		// would change the wire format. It still has to be expressible.
		ref, err := g.Resolve(n.GetBase())
		if err == nil {
			_, err = g.Expand(ref)
		}
		if err != nil {
			return "", err
		}
		g.WarnWhere(d)
		return "", nil
	}

	var b strings.Builder
	var declared []string
	var err error
	switch e := d.GetEnumeration(); {
	case e == nil:
		declared, err = g.message(&b, d)
	case emit.Fielded(e):
		declared, err = g.sum(&b, d)
	default:
		declared, err = g.enum(&b, d)
	}
	if err != nil {
		return "", err
	}

	for _, n := range declared {
		if other, ok := g.names[n]; ok {
			return "", emit.Unsupported(pos, "%s would declare %s in protobuf, and %s already does", name, n, other)
		}
	}
	for _, n := range declared {
		g.names[n] = name
	}
	g.WarnConstraints(d)
	return b.String(), nil
}

// name is what a `name` directive renames a node to, held to protobuf's
// identifier rule, or fallback when the node carries none. A value the
// protobuf compiler would refuse is reported rather than written out, since
// nothing downstream reads the file again to catch it.
func (g *generator) name(all []*ir.Directive, fallback string) (string, error) {
	d, ok := g.Find(all, "name")
	if !ok {
		return fallback, nil
	}
	n := d.GetArgs()[0].GetText()
	if !ident.MatchString(n) {
		return "", emit.Unsupported(d.GetPosition(), "%q is not a protobuf name", n)
	}
	return n, nil
}

// declName is [generator.name] over a declaration, matching what
// [emit.Session.DeclName] would return for a valid one.
func (g *generator) declName(d *ir.Decl) (string, error) {
	return g.name(d.GetDirectives(), emit.Pascal(emit.LastSegment(d.GetMeta().GetName())))
}

// message renders an entity, a value, or a mixin. The three differ in what
// they mean and not in what they emit.
func (g *generator) message(b *strings.Builder, d *ir.Decl) ([]string, error) {
	name, err := g.declName(d)
	if err != nil {
		return nil, err
	}
	comment(b, "", d.GetMeta())
	fmt.Fprintf(b, "message %s {\n", name)
	numbers, names := g.reserved(b, d)
	g.declOptions(b, d.GetMeta(), d.GetDirectives())
	slots, err := g.fields(b, "  ", name, d.Fields(), nil, numbers)
	if err != nil {
		return nil, err
	}
	if err := g.checkReserved(name, slots, numbers, names); err != nil {
		return nil, err
	}
	b.WriteString("}\n")
	return []string{name}, nil
}

// reserved writes one `reserved` statement per directive on d and returns
// the field numbers and names they reserve.
func (g *generator) reserved(b *strings.Builder, d *ir.Decl) (map[int64]bool, map[string]bool) {
	numbers := map[int64]bool{}
	names := map[string]bool{}
	for _, r := range plugin.Directives(g.Target, d.GetDirectives()) {
		if r.GetName() != "reserved" {
			continue
		}
		var args []string
		for _, a := range r.GetArgs() {
			if a.GetKind() == ir.LiteralKind_LITERAL_KIND_STRING {
				names[a.GetText()] = true
				// Editions refuse the quoted form proto3 uses.
				if g.edition != "" {
					args = append(args, a.GetText())
				} else {
					args = append(args, fmt.Sprintf("%q", a.GetText()))
				}
				continue
			}
			if n, err := strconv.ParseInt(a.GetText(), 0, 64); err == nil {
				numbers[n] = true
			}
			args = append(args, a.GetText())
		}
		fmt.Fprintf(b, "  reserved %s;\n", strings.Join(args, ", "))
	}
	return numbers, names
}

// checkReserved refuses a field, or an inlined oneof's member, whose number
// or name the message reserves, since protoc rejects such a message. Only a
// pin can land on a reserved number, since unpinned members skip them. slots
// is what [generator.fields] gave out.
func (g *generator) checkReserved(owner string, slots []slot, numbers map[int64]bool, names map[string]bool) error {
	for _, s := range slots {
		if numbers[s.num] {
			pos := s.Position
			if p, ok := g.Find(s.Directives, "number"); ok {
				pos = p.GetPosition()
			}
			return emit.Unsupported(pos, "%s.%s is numbered %d, which the message reserves", owner, s.Name, s.num)
		}
		if names[s.name] {
			return emit.Unsupported(s.Position, "%s.%s is named %s, which the message reserves", owner, s.Name, s.name)
		}
	}
	return nil
}

// enum renders an enum whose variants carry no fields.
func (g *generator) enum(b *strings.Builder, d *ir.Decl) ([]string, error) {
	name, err := g.declName(d)
	if err != nil {
		return nil, err
	}
	variants := d.GetEnumeration().GetVariants()
	nums, err := g.Numbers(name, emit.VariantMembers(variants), enumNumbers)
	if err != nil {
		return nil, err
	}

	// Enum values share the package's scope, which is why each carries the
	// enum's name.
	prefix := emit.ScreamingSnake(name)
	values := []string{prefix + "_UNSPECIFIED"}

	var body strings.Builder
	fmt.Fprintf(&body, "  %s = 0;\n", values[0])
	for i, v := range variants {
		value, err := g.name(v.GetDirectives(), prefix+"_"+emit.ScreamingSnake(v.GetMeta().GetName()))
		if err != nil {
			return nil, err
		}
		if slices.Contains(values, value) {
			return nil, emit.Unsupported(v.GetMeta().GetPosition(), "%s has two values named %s in protobuf", name, value)
		}
		values = append(values, value)

		comment(&body, "  ", v.GetMeta())
		fmt.Fprintf(&body, "  %s = %d%s;\n", value, nums[i], brackets(g.options(v.GetMeta(), v.GetDirectives())))
	}

	comment(b, "", d.GetMeta())
	fmt.Fprintf(b, "enum %s {\n", name)
	g.declOptions(b, d.GetMeta(), d.GetDirectives())
	b.WriteString(body.String())
	b.WriteString("}\n")
	return append([]string{name}, values...), nil
}

// sum renders an enum where any variant carries fields: a message holding
// a oneof, with a nested message per variant.
//
// A variant with no fields still gets a message, empty, since a oneof
// member is a field and a field has a type.
func (g *generator) sum(b *strings.Builder, d *ir.Decl) ([]string, error) {
	name, err := g.declName(d)
	if err != nil {
		return nil, err
	}
	variants := d.GetEnumeration().GetVariants()
	nums, err := g.Numbers(name, emit.VariantMembers(variants), fieldNumbers)
	if err != nil {
		return nil, err
	}

	nested := map[string]bool{}
	messages := make([]string, len(variants))
	fields := make([]string, len(variants))
	taken := map[string]bool{oneof: true}
	for i, v := range variants {
		pos := v.GetMeta().GetPosition()
		messages[i], err = g.name(v.GetDirectives(), emit.Pascal(v.GetMeta().GetName()))
		if err != nil {
			return nil, err
		}
		if nested[messages[i]] {
			return nil, emit.Unsupported(pos, "%s has two variants named %s in protobuf", name, messages[i])
		}
		nested[messages[i]] = true

		fields[i] = emit.Snake(v.GetMeta().GetName())
		if taken[fields[i]] {
			return nil, emit.Unsupported(pos, "%s would have two members named %s in protobuf", name, fields[i])
		}
		taken[fields[i]] = true
	}

	comment(b, "", d.GetMeta())
	fmt.Fprintf(b, "message %s {\n", name)
	g.declOptions(b, d.GetMeta(), d.GetDirectives())
	for i, v := range variants {
		comment(b, "  ", v.GetMeta())
		fmt.Fprintf(b, "  message %s {\n", messages[i])
		if _, err := g.fields(b, "    ", name+"."+messages[i], v.GetFields(), nested, nil); err != nil {
			return nil, err
		}
		b.WriteString("  }\n")
	}
	fmt.Fprintf(b, "  oneof %s {\n", oneof)
	for i, v := range variants {
		// A oneof member carries its variant's deprecation and none of its
		// option directives.
		fmt.Fprintf(b, "    %s %s = %d%s;\n", messages[i], fields[i], nums[i], brackets(g.options(v.GetMeta(), nil)))
	}
	b.WriteString("  }\n}\n")
	return []string{name}, nil
}

// slot is a number a message body gives out, to a field or to one member of
// an inlined oneof, which protobuf counts as a field of the message.
type slot struct {
	emit.Member
	name string // the protobuf name written
	num  int64
}

// fields renders a message body and returns the numbers it gave out, in the
// order written. nested is the names of the messages declared beside it,
// which a reference has to step around; skip is the numbers unpinned fields
// and oneof members pass over.
func (g *generator) fields(b *strings.Builder, indent, owner string, fields []*ir.Field, nested map[string]bool, skip map[int64]bool) ([]slot, error) {
	// An inlined oneof's variants take numbers in the message's space in
	// the oneof's place, and the oneof takes none.
	var members []emit.Member
	first := make([]int, len(fields))
	for i, f := range fields {
		first[i] = len(members)
		if !g.isOneof(f) {
			members = append(members, emit.FieldMembers(fields[i:i+1])...)
			continue
		}
		ref, err := g.Resolve(f.GetType())
		if err != nil {
			return nil, err
		}
		if err := g.inlinable(f, ref); err != nil {
			return nil, err
		}
		members = append(members, emit.VariantMembers(ref.Decl.GetEnumeration().GetVariants())...)
	}
	rule := fieldNumbers
	rule.Skip = skip
	nums, err := g.Numbers(owner, members, rule)
	if err != nil {
		return nil, err
	}

	seen := map[string]bool{}
	out := make([]slot, 0, len(members))
	for i, f := range fields {
		ref, err := g.Resolve(f.GetType())
		if err != nil {
			return nil, err
		}
		if g.isOneof(f) {
			if err := g.inlineOneof(b, indent, f, ref, nums[first[i]:], nested); err != nil {
				return nil, err
			}
			for j, v := range ref.Decl.GetEnumeration().GetVariants() {
				k := first[i] + j
				out = append(out, slot{members[k], emit.Snake(v.GetFields()[0].GetMeta().GetName()), nums[k]})
			}
			continue
		}
		label, typ, err := g.fieldType(ref, nested)
		if err != nil {
			return nil, err
		}

		name, err := g.name(f.GetDirectives(), emit.Snake(f.GetMeta().GetName()))
		if err != nil {
			return nil, err
		}
		if seen[name] {
			return nil, emit.Unsupported(f.GetMeta().GetPosition(), "%s has two fields named %s in protobuf", owner, name)
		}
		seen[name] = true

		comment(b, indent, f.GetMeta())
		if label != "" {
			label += " "
		}
		fmt.Fprintf(b, "%s%s%s %s = %d%s;\n", indent, label, typ, name, nums[first[i]], brackets(g.options(f.GetMeta(), f.GetDirectives())))
		out = append(out, slot{members[first[i]], name, nums[first[i]]})
	}
	return out, nil
}

// oneofDirective is a field's `oneof` directive for this target, or nil.
// [emit.Session.Find] skips a directive that takes no argument.
func (g *generator) oneofDirective(f *ir.Field) *ir.Directive {
	for _, d := range plugin.Directives(g.Target, f.GetDirectives()) {
		if d.GetName() == "oneof" {
			return d
		}
	}
	return nil
}

func (g *generator) isOneof(f *ir.Field) bool { return g.oneofDirective(f) != nil }

// inlinedOnly is the sum types every use of which is a field carrying
// `oneof`. Each such field writes the variants out itself, so the sum
// type's message would be declared and never named.
func (g *generator) inlinedOnly() map[*ir.Decl]bool {
	inlined := map[*ir.Decl]bool{}
	var named []*ir.ID
	for _, d := range g.Model.GetDecls() {
		if n := d.GetNewtype(); n != nil {
			named = append(named, n.GetBase())
			continue
		}
		fields := d.Fields()
		for _, v := range d.GetEnumeration().GetVariants() {
			fields = append(fields, v.GetFields()...)
		}
		for _, f := range fields {
			if !g.isOneof(f) {
				named = append(named, f.GetType())
			} else if ref, err := g.Resolve(f.GetType()); err == nil && ref.Form == emit.Named {
				inlined[ref.Decl] = true
			}
		}
	}
	for _, d := range g.TypeReferences(named...) {
		delete(inlined, d)
	}
	return inlined
}

// inlinable reports why a field's `oneof` directive cannot be honored: its
// type has to be an enum whose every variant carries exactly one field.
func (g *generator) inlinable(f *ir.Field, ref *emit.Ref) error {
	pos := g.oneofDirective(f).GetPosition()
	name := f.GetMeta().GetName()
	e := ref.Decl.GetEnumeration()
	if ref.Form != emit.Named || e == nil {
		return emit.Unsupported(pos, "field %s is a oneof, and only an enum can be inlined as one", name)
	}
	for _, v := range e.GetVariants() {
		if n := len(v.GetFields()); n != 1 {
			return emit.Unsupported(pos, "field %s is a oneof, and variant %s carries %d fields rather than one", name, v.GetMeta().GetName(), n)
		}
	}
	return nil
}

// inlineOneof renders a field whose type is a sum type as a oneof of the
// variants' single fields, numbered by nums in variant order.
func (g *generator) inlineOneof(b *strings.Builder, indent string, f *ir.Field, ref *emit.Ref, nums []int64, nested map[string]bool) error {
	variants := ref.Decl.GetEnumeration().GetVariants()
	fmt.Fprintf(b, "%soneof %s {\n", indent, emit.Snake(f.GetMeta().GetName()))
	for i, v := range variants {
		vf := v.GetFields()[0]
		r, err := g.Resolve(vf.GetType())
		if err != nil {
			return err
		}
		typ, err := g.single(r, nested)
		if err != nil {
			return err
		}
		// The member stands for both the variant and its one field, so it
		// carries the options of each and is deprecated when either is.
		meta := vf.GetMeta()
		if v.GetMeta().IsDeprecated() {
			meta = v.GetMeta()
		}
		opts := g.options(meta, append(slices.Clone(v.GetDirectives()), vf.GetDirectives()...))
		fmt.Fprintf(b, "%s  %s %s = %d%s;\n", indent, typ, emit.Snake(vf.GetMeta().GetName()), nums[i], brackets(opts))
	}
	fmt.Fprintf(b, "%s}\n", indent)
	return nil
}

// fieldType returns a field's label and type.
//
// A repeated or map field cannot be optional and cannot hold another
// repeated or map field, so each of those shapes is reported rather than
// flattened into something that means less.
func (g *generator) fieldType(r *emit.Ref, nested map[string]bool) (label, typ string, err error) {
	if r, err = g.Expand(r); err != nil {
		return "", "", err
	}

	switch r.Form {
	case emit.Option, emit.Nullable:
		inner, err := g.Expand(r.Elem)
		if err != nil {
			return "", "", err
		}
		if !single(inner) {
			return "", "", emit.Unsupported(r.Pos, "%s holding %s has no protobuf form", describe(r), describe(inner))
		}
		typ, err := g.single(inner, nested)
		if err != nil {
			return "", "", err
		}
		// A message field already has presence, and `optional` on one
		// says nothing more. Under an edition every field has explicit
		// presence by default.
		if g.edition != "" || isMessage(inner) {
			return "", typ, nil
		}
		return "optional", typ, nil

	case emit.List, emit.Set:
		// Protobuf has no set, and a repeated field does not keep one's
		// guarantee. docs/design/schema-backends.md says so.
		elem, err := g.Expand(r.Elem)
		if err != nil {
			return "", "", err
		}
		if !single(elem) {
			return "", "", emit.Unsupported(r.Pos, "%s holding %s has no protobuf form", describe(r), describe(elem))
		}
		typ, err := g.single(elem, nested)
		return "repeated", typ, err

	case emit.Map:
		key, err := g.Expand(r.Key)
		if err != nil {
			return "", "", err
		}
		val, err := g.Expand(r.Elem)
		if err != nil {
			return "", "", err
		}
		if !single(val) {
			return "", "", emit.Unsupported(r.Pos, "%s holding %s has no protobuf form", describe(r), describe(val))
		}
		k, err := g.single(key, nested)
		if err != nil {
			return "", "", err
		}
		if key.Form != emit.Prim || !mapKeys[k] {
			return "", "", emit.Unsupported(r.Pos, "a protobuf map key is a string, an integer, or a bool, and this one is %s", k)
		}
		v, err := g.single(val, nested)
		return "", "map<" + k + ", " + v + ">", err
	}

	typ, err = g.single(r, nested)
	return "", typ, err
}

// single is the protobuf type for a primitive or a declaration.
func (g *generator) single(r *emit.Ref, nested map[string]bool) (string, error) {
	if r.Form == emit.Prim {
		typ, ok := scalars[r.Name]
		if !ok {
			return "", emit.Unsupported(r.Pos, "primitive %s has no protobuf type", r.Name)
		}
		if imp, ok := wellKnown[typ]; ok {
			g.imports[imp] = true
		}
		return typ, nil
	}

	name := g.DeclName(r.Decl, emit.Pascal)
	// Protobuf resolves a name from the innermost scope out, so inside a
	// sum type a declaration sharing a variant's name would resolve to the
	// variant. A fully qualified name resolves from the root.
	if nested[name] {
		if g.pkg == "" {
			return "." + name, nil
		}
		return "." + g.pkg + "." + name, nil
	}
	return name, nil
}

func single(r *emit.Ref) bool { return r.Form == emit.Prim || r.Form == emit.Named }

func isMessage(r *emit.Ref) bool {
	if r.Form != emit.Named {
		return false
	}
	e := r.Decl.GetEnumeration()
	return e == nil || emit.Fielded(e)
}

// describe names a shape for a message.
func describe(r *emit.Ref) string {
	switch r.Form {
	case emit.List:
		return "a list"
	case emit.Set:
		return "a set"
	case emit.Map:
		return "a map"
	case emit.Option:
		return "an optional value"
	case emit.Nullable:
		return "a nullable value"
	}
	return "a value"
}

// comment writes a node's documentation, and its deprecation reason when
// it gave one.
func comment(b *strings.Builder, indent string, meta *ir.Meta) {
	lines := emit.Doc(meta)
	if reason, ok := emit.Deprecated(meta); ok && reason != "" {
		if len(lines) > 0 {
			lines = append(lines, "")
		}
		lines = append(lines, "Deprecated: "+reason)
	}
	for _, line := range lines {
		if line == "" {
			fmt.Fprintf(b, "%s//\n", indent)
			continue
		}
		fmt.Fprintf(b, "%s// %s\n", indent, line)
	}
}

// options is a node's options as `name = value`: `deprecated = true` when
// the node is deprecated, then each `option` directive in source order. An
// option directive naming deprecated is dropped when the node is already
// deprecated, since protoc refuses an option set twice.
func (g *generator) options(meta *ir.Meta, dirs []*ir.Directive) []string {
	var opts []string
	if meta.IsDeprecated() {
		opts = append(opts, "deprecated = true")
	}
	for _, d := range plugin.Directives(g.Target, dirs) {
		if d.GetName() != "option" || len(d.GetArgs()) != 2 {
			continue
		}
		if meta.IsDeprecated() && d.GetArgs()[0].GetText() == "deprecated" {
			continue
		}
		opts = append(opts, d.GetArgs()[0].GetText()+" = "+d.GetArgs()[1].GetText())
	}
	return opts
}

// declOptions writes a message's or an enum's options as statements.
func (g *generator) declOptions(b *strings.Builder, meta *ir.Meta, dirs []*ir.Directive) {
	for _, o := range g.options(meta, dirs) {
		fmt.Fprintf(b, "  option %s;\n", o)
	}
}

// brackets is a field's or an enum value's options as a bracket list.
func brackets(opts []string) string {
	if len(opts) == 0 {
		return ""
	}
	return " [" + strings.Join(opts, ", ") + "]"
}

func validPackage(pkg string) bool {
	if pkg == "" {
		return true
	}
	for seg := range strings.SplitSeq(pkg, ".") {
		if !ident.MatchString(seg) {
			return false
		}
	}
	return true
}

// filePath places the file where buf expects a package's files to be: in
// the directories its name spells. The file directive names the file, and
// the package's last segment with .proto is the default.
func (g *generator) filePath() string {
	name := "model.proto"
	if g.pkg != "" {
		name = emit.LastSegment(g.pkg) + ".proto"
	}
	if g.file != "" {
		name = g.file
	}
	return path.Join(strings.ReplaceAll(g.pkg, ".", "/"), name)
}
