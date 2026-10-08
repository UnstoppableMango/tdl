// Package protobuf generates a proto3 schema, or one under the edition an
// `edition` directive names, from a resolved model. The mapping is in
// docs/design/schema-backends.md.
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

	"github.com/unstoppablemango/tdl/ast"
	"github.com/unstoppablemango/tdl/backend/internal/emit"
	"github.com/unstoppablemango/tdl/internal/unlower"
	"github.com/unstoppablemango/tdl/ir"
	"github.com/unstoppablemango/tdl/plugin"
)

// Name names this backend in a target block and as tdl-gen-protobuf.
const Name = "protobuf"

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
			// The dotted protobuf package; defaults to the model's.
			{Name: "package", MinArgs: 1, MaxArgs: 1, ArgKinds: str},
			// The file name, in place of the package's last segment + .proto.
			{Name: "file", MinArgs: 1, MaxArgs: 1, ArgKinds: str},
			// The protobuf name for a declaration, field, enum value, or
			// variant message.
			{Name: "name", MinArgs: 1, MaxArgs: 1, ArgKinds: str},
			// The wire number of a field, variant, or enum value.
			{Name: "number", MinArgs: 1, MaxArgs: 1, ArgKinds: []ir.LiteralKind{ir.LiteralKind_LITERAL_KIND_INT}},
			// The edition the file declares in place of proto3.
			{Name: "edition", MinArgs: 1, MaxArgs: 1, ArgKinds: str},
			// Numbers or names a message reserves. arg_kinds is per position
			// and cannot say "int or string", so it is unset.
			{Name: "reserved", MinArgs: 1, MaxArgs: -1, Repeatable: true},
			// Inlines a field's sum type into its message as a oneof.
			{Name: "oneof"},
			// A file every output file imports.
			{Name: "import", MinArgs: 1, MaxArgs: 1, ArgKinds: str, Repeatable: true},
			// `name = value`, in a field's or enum value's brackets, or as an
			// `option` statement in a message, enum, service, or rpc.
			{Name: "option", MinArgs: 2, MaxArgs: 2, ArgKinds: []ir.LiteralKind{ir.LiteralKind_LITERAL_KIND_STRING, ir.LiteralKind_LITERAL_KIND_STRING}, Repeatable: true},
			// A message another proto file declares: the file to import and
			// the message's full name. The declaration itself is not emitted.
			{Name: "foreign", MinArgs: 2, MaxArgs: 2, ArgKinds: append(str, str...)},
			// Emits a structure as a service.
			{Name: "service"},
			// Tags a primitive of two type arguments (request, response)
			// that a service field applies to declare an rpc.
			{Name: "rpc"},
			// Tags a primitive of one type argument marking an rpc's request
			// or response streamed.
			{Name: "stream"},
			// Writes, as custom options from tdl/annotations.proto, every
			// fact the schema alone would lose, so import rebuilds the model.
			{Name: "roundtrip"},
		},
		Reverse: true,
	}
}

var (
	// fieldNumbers covers fields and oneof members, which share a number space.
	fieldNumbers = emit.NumberRule{Max: 536870911, Reserved: [][2]int64{{19000, 19999}}}
	// enumNumbers leaves zero to the UNSPECIFIED value protobuf requires first.
	enumNumbers = emit.NumberRule{Max: math.MaxInt32}
)

// scalars maps a prelude primitive to its protobuf type.
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

var wellKnown = map[string]string{
	"google.protobuf.Timestamp": "google/protobuf/timestamp.proto",
	"google.protobuf.Duration":  "google/protobuf/duration.proto",
}

var mapKeys = map[string]bool{
	"string": true, "bool": true,
	"int32": true, "int64": true, "uint32": true, "uint64": true,
}

var editions = map[string]bool{"2023": true, "2024": true}

// oneof names the oneof in a sum type's message.
const oneof = "variant"

var ident = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

type generator struct {
	*emit.Session
	pkg string
	// file is the `file` directive's argument, or "".
	file string

	// edition is "" for proto3.
	edition string

	// names maps each package-scope name, enum values included, to its
	// declaration.
	names map[string]string

	// imports is what the declaration being rendered needs.
	imports map[string]bool
	// refs is the declarations it names.
	refs map[*ir.Decl]bool

	// roundtrip writes annotations in place of loss warnings.
	roundtrip bool
	// multi is set when the declarations are placed in more than one file.
	multi bool
	// annotated is set when the declaration being rendered wrote an
	// annotation.
	annotated bool
	// source is the model unlowered, read for the TDL an annotation
	// carries.
	source *ast.File
}

// rendered is one declaration's text, held until the cascade decides
// whether it is emitted.
type rendered struct {
	decl      *ir.Decl
	path      string
	text      string
	imports   map[string]bool
	refs      map[*ir.Decl]bool
	annotated bool
}

// Generate returns one .proto file per file the declarations are placed in.
func (Backend) Generate(_ context.Context, req *plugin.Request) (*plugin.Response, error) {
	g := &generator{
		Session: emit.NewSession(req, "protobuf"),
		pkg:     req.GetModel().GetPackage(),
		names:   map[string]string{},
	}
	g.Externs = true

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
	// An invalid package fails the whole output.
	if !validPackage(g.pkg) {
		g.Error(pkgPos, "%q is not a protobuf package name", g.pkg)
		return g.Response(nil), nil
	}
	if d, ok := g.Block("file"); ok {
		g.file = d.GetArgs()[0].GetText()
		if !validFile(g.file) {
			g.Error(d.GetPosition(), "%q is not a protobuf file name", g.file)
			return g.Response(nil), nil
		}
	}

	g.roundtrip = g.bare("roundtrip")
	if g.pkg != req.GetModel().GetPackage() {
		g.lose(emit.LossName, pkgPos, "package %s is written %s, which reads back as the package", req.GetModel().GetPackage(), g.pkg)
	}
	if len(req.GetModel().GetDoc()) > 0 && g.pkg == "" {
		g.Lossy(emit.LossDoc, nil, "the package's doc comment has no package statement to precede")
	}

	own := g.Own()
	inlinedOnly := g.inlinedOnly()
	g.multi = g.spansFiles(own)
	skipped := map[*ir.Decl]bool{}
	var out []rendered
	// Each declaration's warnings are held until the cascade decides
	// whether it is emitted, since a skipped one loses everything at once.
	held := map[*ir.Decl][]*plugin.Diagnostic{}
	for _, d := range own {
		if inlinedOnly[d] {
			continue
		}
		g.imports = map[string]bool{}
		g.refs = map[*ir.Decl]bool{}
		g.annotated = false
		mark := len(g.Diags)
		path, err := g.pathOf(d)
		if err != nil {
			g.Diags = g.Diags[:mark]
			g.skip(err)
			skipped[d] = true
			continue
		}
		text, err := g.decl(d)
		if err != nil {
			g.Diags = g.Diags[:mark]
			g.skip(err)
			skipped[d] = true
			continue
		}
		held[d] = slices.Clone(g.Diags[mark:])
		g.Diags = g.Diags[:mark]
		if text != "" {
			out = append(out, rendered{d, path, text, g.imports, g.refs, g.annotated})
		}
	}
	mark := len(g.Diags)
	g.Cascade(own, skipped)
	if g.roundtrip {
		// A skipped declaration is carried as TDL, so nothing is lost.
		g.Diags = g.Diags[:mark]
	}
	for _, d := range own {
		if !skipped[d] {
			g.Diags = append(g.Diags, held[d]...)
		}
	}

	placed := map[*ir.Decl]string{}
	for _, r := range out {
		if !skipped[r.decl] {
			placed[r.decl] = r.path
		}
	}

	groups := map[string]*group{}
	add := func(path string) *group {
		grp := groups[path]
		if grp == nil {
			grp = &group{imports: map[string]bool{}}
			groups[path] = grp
		}
		return grp
	}
	for _, r := range out {
		if skipped[r.decl] {
			continue
		}
		grp := add(r.path)
		grp.blocks = append(grp.blocks, r.text)
		maps.Copy(grp.imports, r.imports)
		for ref := range r.refs {
			if other, ok := placed[ref]; ok && other != r.path {
				grp.imports[other] = true
			}
		}
		if r.annotated {
			grp.imports[annotationsFile] = true
		}
	}
	if g.roundtrip {
		if ann := g.fileAnnotation(placed); ann != "" {
			first := filePath(g.pkg, g.file)
			if len(groups) > 0 {
				first = slices.Sorted(maps.Keys(groups))[0]
			}
			grp := add(first)
			grp.annotation = ann
			grp.imports[annotationsFile] = true
		}
	}
	for _, block := range req.GetModel().GetTargets() {
		if block.GetMeta().GetName() != g.Target {
			continue
		}
		opts := g.options(nil, block.GetDirectives())
		for _, grp := range groups {
			for _, d := range plugin.Directives(g.Target, block.GetDirectives()) {
				if d.GetName() == "import" && len(d.GetArgs()) > 0 {
					grp.imports[d.GetArgs()[0].GetText()] = true
				}
			}
			grp.options = append(grp.options, opts...)
		}
	}
	if len(groups) == 0 {
		return g.Response(nil), nil
	}

	var files []*plugin.File
	annotations := false
	for _, p := range slices.Sorted(maps.Keys(groups)) {
		files = append(files, g.write(p, groups[p]))
		annotations = annotations || groups[p].imports[annotationsFile]
	}
	if annotations {
		files = append(files, &plugin.File{Path: annotationsFile, Content: annotationsProto})
	}
	return g.Response(files), nil
}

// group is the declarations placed in one file, what they import, and the
// file options.
type group struct {
	blocks  []string
	imports map[string]bool
	options []string
	// annotation is the file's (tdl.file) option, or "".
	annotation string
}

func (g *generator) write(path string, grp *group) *plugin.File {
	header := `syntax = "proto3";`
	if g.edition != "" {
		header = fmt.Sprintf("edition = %q;", g.edition)
	}
	var b strings.Builder
	fmt.Fprintf(&b, "// Code generated by tdl. DO NOT EDIT.\n\n%s\n", header)
	if g.pkg != "" {
		b.WriteString("\n")
		commentLines(&b, "", emit.Doc(&ir.Meta{Doc: g.Model.GetDoc()}))
		fmt.Fprintf(&b, "package %s;\n", g.pkg)
	}
	if len(grp.imports) > 0 {
		b.WriteString("\n")
		for _, imp := range slices.Sorted(maps.Keys(grp.imports)) {
			fmt.Fprintf(&b, "import %q;\n", imp)
		}
	}
	if len(grp.options) > 0 || grp.annotation != "" {
		b.WriteString("\n")
		for _, o := range grp.options {
			fmt.Fprintf(&b, "option %s;\n", o)
		}
		if grp.annotation != "" {
			fmt.Fprintf(&b, "option %s;\n", grp.annotation)
		}
	}
	for _, block := range grp.blocks {
		b.WriteString("\n")
		b.WriteString(block)
	}
	return &plugin.File{Path: path, Content: []byte(b.String())}
}

// pathOf is the file a declaration is placed in: its `file` directive's,
// else the model's.
func (g *generator) pathOf(d *ir.Decl) (string, error) {
	fd, ok := g.Find(d.GetDirectives(), "file")
	if !ok {
		return filePath(g.pkg, g.file), nil
	}
	name := fd.GetArgs()[0].GetText()
	if !validFile(name) {
		return "", emit.Unsupported(fd.GetPosition(), "%q is not a protobuf file name", name)
	}
	return filePath(g.pkg, name), nil
}

// decl renders one declaration, or "" for one that declares nothing.
func (g *generator) decl(d *ir.Decl) (string, error) {
	pos := d.GetMeta().GetPosition()
	name := d.GetMeta().GetName()

	if _, ok := g.Find(d.GetDirectives(), "foreign"); ok {
		return "", nil
	}

	switch {
	case d.GetClass() != nil:
		return "", emit.Lost(emit.LossClass, pos, "%s is a class, and classes are not generated yet", name)
	case d.GetUnit() != nil:
		return "", emit.Lost(emit.LossUnit, pos, "%s is a unit, and units are not generated yet", name)
	case d.GetAlias() != nil:
		if len(d.Params()) == 0 {
			g.lose(emit.LossAlias, pos, "alias %s is expanded where used", name)
			g.typeLosses(d.GetAlias().GetTarget(), nil)
		}
		return "", nil
	case d.GetStructure() == nil && d.GetEnumeration() == nil && d.GetNewtype() == nil:
		// A primitive declares nothing.
		return "", nil
	}
	if len(d.Params()) > 0 {
		return "", emit.Lost(emit.LossGeneric, pos, "%s is parameterized, and generics are not generated yet", name)
	}

	if n := d.GetNewtype(); n != nil {
		// A newtype is expanded where used, since a wrapper message would
		// change the wire format. Its base must still be expressible.
		ref, err := g.Resolve(n.GetBase())
		if err == nil {
			_, err = g.Expand(ref)
		}
		if err != nil {
			return "", err
		}
		g.lose(emit.LossNewtype, pos, "newtype %s is expanded to its base where used", name)
		g.typeLosses(n.GetBase(), nil)
		if !g.roundtrip {
			g.WarnWhere(d)
		}
		return "", nil
	}

	var b strings.Builder
	var declared []string
	var err error
	switch e := d.GetEnumeration(); {
	case e == nil && g.tagged(d.GetDirectives(), "service"):
		declared, err = g.service(&b, d)
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
	if !g.roundtrip {
		g.WarnConstraints(d)
	}
	return b.String(), nil
}

// name is a node's `name` directive, held to protobuf's identifier rule, or
// fallback when it has none.
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

// declName is [generator.name] over a declaration.
func (g *generator) declName(d *ir.Decl) (string, error) {
	return g.name(d.GetDirectives(), emit.Pascal(emit.LastSegment(d.GetMeta().GetName())))
}

// message renders an entity, a value, or a mixin, which emit alike.
func (g *generator) message(b *strings.Builder, d *ir.Decl) ([]string, error) {
	name, err := g.declName(d)
	if err != nil {
		return nil, err
	}
	comment(b, "", d.GetMeta())
	fmt.Fprintf(b, "message %s {\n", name)
	numbers, names := g.reserved(b, d)
	g.declOptions(b, d.GetMeta(), d.GetDirectives(), g.structAnnotation(d, name))
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

// checkReserved refuses a slot whose number or name the message reserves,
// since protoc rejects it. Only a pin can land on a reserved number.
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

// service renders a structure tagged `service`: each field is an rpc,
// typed by a primitive tagged `rpc` applied to the request and response.
func (g *generator) service(b *strings.Builder, d *ir.Decl) ([]string, error) {
	name, err := g.declName(d)
	if err != nil {
		return nil, err
	}
	comment(b, "", d.GetMeta())
	fmt.Fprintf(b, "service %s {\n", name)
	g.declOptions(b, d.GetMeta(), d.GetDirectives(), g.serviceAnnotation(d))
	for _, f := range d.Fields() {
		t := g.Model.Type(f.GetType())
		if !g.ctorTagged(t, "rpc") || len(t.GetArgs()) != 2 {
			return nil, emit.Unsupported(f.GetMeta().GetPosition(), "%s is a service, and its field %s is not an rpc", name, f.GetMeta().GetName())
		}
		req, err := g.rpcArg(f, t.GetArgs()[0])
		if err != nil {
			return nil, err
		}
		res, err := g.rpcArg(f, t.GetArgs()[1])
		if err != nil {
			return nil, err
		}
		comment(b, "  ", f.GetMeta())
		end := ";"
		if opts := g.options(f.GetMeta(), f.GetDirectives()); len(opts) > 0 {
			end = " { option " + strings.Join(opts, "; option ") + "; }"
		}
		fmt.Fprintf(b, "  rpc %s(%s) returns (%s)%s\n", f.GetMeta().GetName(), req, res, end)
	}
	b.WriteString("}\n")
	return []string{name}, nil
}

// rpcArg renders an rpc's request or response: a message, prefixed
// `stream ` when a primitive tagged `stream` wraps it.
func (g *generator) rpcArg(f *ir.Field, arg *ir.ID) (string, error) {
	prefix := ""
	if t := g.Model.Type(arg); len(t.GetArgs()) == 1 && g.ctorTagged(t, "stream") {
		prefix, arg = "stream ", t.GetArgs()[0]
	}
	ref, err := g.Resolve(arg)
	if err == nil {
		ref, err = g.Expand(ref)
	}
	if err != nil {
		return "", err
	}
	if !isMessage(ref) {
		return "", emit.Unsupported(f.GetMeta().GetPosition(), "an rpc's request and response are messages, and %s's are not", f.GetMeta().GetName())
	}
	typ, err := g.single(ref, nil)
	return prefix + typ, err
}

// ctorTagged reports whether t applies a primitive tagged name, declared
// locally or imported as an extern.
func (g *generator) ctorTagged(t *ir.Type, name string) bool {
	if e := t.GetExtern(); e.Resolved() && int(e.GetIndex()) < len(g.Model.GetExterns()) {
		return g.tagged(g.Model.GetExterns()[e.GetIndex()].GetDirectives(), name)
	}
	c := g.Model.Decl(t.GetCtor())
	return c.GetPrimitive() != nil && g.tagged(c.GetDirectives(), name)
}

// tagged reports whether a node carries an argument-less directive.
func (g *generator) tagged(all []*ir.Directive, name string) bool {
	for _, d := range plugin.Directives(g.Target, all) {
		if d.GetName() == name {
			return true
		}
	}
	return false
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
	g.checkPins(name, emit.VariantMembers(variants), nums, enumNumbers)

	// Enum values have package scope, so each carries the enum's name.
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

		opts := g.options(v.GetMeta(), v.GetDirectives())
		if back, renamed := valueBack(prefix, value); back != v.GetMeta().GetName() {
			g.lose(emit.LossName, v.GetMeta().GetPosition(), "%s.%s is written %s, which reads back as %s", name, v.GetMeta().GetName(), value, back)
			opts = g.annotate(opts, "value", text("name", v.GetMeta().GetName()))
		} else {
			g.checkRenamed(v.GetDirectives(), renamed, "%s.%s", name, v.GetMeta().GetName())
		}
		comment(&body, "  ", v.GetMeta())
		fmt.Fprintf(&body, "  %s = %d%s;\n", value, nums[i], brackets(opts))
	}

	comment(b, "", d.GetMeta())
	fmt.Fprintf(b, "enum %s {\n", name)
	g.declOptions(b, d.GetMeta(), d.GetDirectives(), g.declAnnotation("enum", d, name))
	b.WriteString(body.String())
	b.WriteString("}\n")
	return append([]string{name}, values...), nil
}

// sum renders an enum where any variant carries fields: a message holding a
// oneof of one nested message per variant, empty for a fieldless variant.
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
	g.checkPins(name, emit.VariantMembers(variants), nums, fieldNumbers)

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
	g.declOptions(b, d.GetMeta(), d.GetDirectives(), g.declAnnotation("message", d, name))
	for i, v := range variants {
		comment(b, "  ", v.GetMeta())
		fmt.Fprintf(b, "  message %s {\n", messages[i])
		if messages[i] != v.GetMeta().GetName() {
			g.lose(emit.LossName, v.GetMeta().GetPosition(), "%s.%s is written %s, which reads back as the variant's name", name, v.GetMeta().GetName(), messages[i])
			if a := g.annotation("message", text("name", v.GetMeta().GetName())); a != "" {
				fmt.Fprintf(b, "    option %s;\n", a)
			}
		} else {
			g.checkRenamed(v.GetDirectives(), emit.Pascal(messages[i]) != messages[i], "%s.%s", name, v.GetMeta().GetName())
		}
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

// slot is a number a message body gives out, to a field or an inlined
// oneof member.
type slot struct {
	emit.Member
	name string // the protobuf name written
	num  int64
}

// fields renders a message body and returns the slots it gave out, in
// order. nested is the sibling message names a reference must step around;
// skip is the numbers unpinned members pass over.
func (g *generator) fields(b *strings.Builder, indent, owner string, fields []*ir.Field, nested map[string]bool, skip map[int64]bool) ([]slot, error) {
	// An inlined oneof's variants take numbers in the message's space; the
	// oneof takes none.
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
	g.checkPins(owner, members, nums, rule)
	includes := unlower.Includes(g.Model, fields)
	for i, inc := range includes {
		if inc != "" && (i == 0 || includes[i-1] != inc) {
			g.lose(emit.LossInclude, fields[i].GetMeta().GetPosition(), "%s's include of %s is flattened into its fields", owner, inc)
		}
	}

	seen := map[string]bool{}
	out := make([]slot, 0, len(members))
	for i, f := range fields {
		ref, err := g.Resolve(f.GetType())
		if err != nil {
			return nil, err
		}
		if g.isOneof(f) {
			if err := g.inlineOneof(b, indent, owner, f, ref, nums[first[i]:], nested, includes[i]); err != nil {
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

		opts := g.options(f.GetMeta(), f.GetDirectives())
		opts = g.fieldLosses(opts, owner, f, ref, name, includes[i])
		comment(b, indent, f.GetMeta())
		if label != "" {
			label += " "
		}
		fmt.Fprintf(b, "%s%s%s %s = %d%s;\n", indent, label, typ, name, nums[first[i]], brackets(opts))
		out = append(out, slot{members[first[i]], name, nums[first[i]]})
	}
	return out, nil
}

// oneofDirective is a field's `oneof` directive, or nil.
// [emit.Session.Find] skips argument-less directives.
func (g *generator) oneofDirective(f *ir.Field) *ir.Directive {
	for _, d := range plugin.Directives(g.Target, f.GetDirectives()) {
		if d.GetName() == "oneof" {
			return d
		}
	}
	return nil
}

func (g *generator) isOneof(f *ir.Field) bool { return g.oneofDirective(f) != nil }

// inlinedOnly is the sum types used only by fields carrying `oneof`, whose
// message would go unused.
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

// inlineOneof renders a `oneof` field as a oneof of its variants' single
// fields, numbered by nums.
func (g *generator) inlineOneof(b *strings.Builder, indent, owner string, f *ir.Field, ref *emit.Ref, nums []int64, nested map[string]bool, include string) error {
	variants := ref.Decl.GetEnumeration().GetVariants()
	comment(b, indent, f.GetMeta())
	fmt.Fprintf(b, "%soneof %s {\n", indent, emit.Snake(f.GetMeta().GetName()))
	if a := g.oneofLosses(owner, f, ref.Decl, include); a != "" {
		fmt.Fprintf(b, "%s  option %s;\n", indent, a)
	}
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
		commentLines(b, indent+"  ", emit.Doc(v.GetMeta()))
		comment(b, indent+"  ", vf.GetMeta())
		fmt.Fprintf(b, "%s  %s %s = %d%s;\n", indent, typ, emit.Snake(vf.GetMeta().GetName()), nums[i], brackets(opts))
	}
	fmt.Fprintf(b, "%s}\n", indent)
	return nil
}

// fieldType returns a field's label and type. A nested repeated, map, or
// optional shape is reported, since protobuf cannot express it.
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
		// A message field already has presence, and under an edition every
		// field does.
		if g.edition != "" || isMessage(inner) {
			return "", typ, nil
		}
		return "optional", typ, nil

	case emit.List, emit.Set:
		// A repeated field does not keep a set's guarantee.
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

	if r.Form == emit.Extern {
		typ, imp, err := g.externRef(r)
		if err != nil {
			return "", err
		}
		g.imports[imp] = true
		return typ, nil
	}

	if f, ok := g.Find(r.Decl.GetDirectives(), "foreign"); ok {
		g.imports[f.GetArgs()[0].GetText()] = true
		return f.GetArgs()[1].GetText(), nil
	}

	g.refs[r.Decl] = true
	name := g.DeclName(r.Decl, emit.Pascal)
	// Protobuf resolves names innermost first, so inside a sum type a
	// declaration sharing a variant's name needs its fully qualified name.
	if nested[name] {
		if g.pkg == "" {
			return "." + name, nil
		}
		return "." + g.pkg + "." + name, nil
	}
	return name, nil
}

// externRef is the type and import for a declaration in another package:
// its foreign directive's, else what the dependency's protobuf target block
// generates.
func (g *generator) externRef(r *emit.Ref) (typ, imp string, err error) {
	if f, ok := g.Find(r.Extern.GetDirectives(), "foreign"); ok {
		return f.GetArgs()[1].GetText(), f.GetArgs()[0].GetText(), nil
	}
	for _, dep := range g.Model.GetImports() {
		if dep.GetPackage() != r.Extern.GetPackage() || len(plugin.Directives(g.Target, dep.GetDirectives())) == 0 {
			continue
		}
		pkg, _ := g.Text(dep.GetDirectives(), "package")
		if pkg == "" {
			pkg = dep.GetPackage()
		}
		file, _ := g.Text(dep.GetDirectives(), "file")
		return pkg + "." + emit.Pascal(emit.LastSegment(r.Extern.GetName())), filePath(pkg, file), nil
	}
	return "", "", emit.Unsupported(r.Pos, "%s is declared in another package, and foreign types are not generated yet", r.Name)
}

func single(r *emit.Ref) bool {
	return r.Form == emit.Prim || r.Form == emit.Named || r.Form == emit.Extern
}

func isMessage(r *emit.Ref) bool {
	// A foreign mapping names a message.
	if r.Form == emit.Extern {
		return true
	}
	if r.Form != emit.Named {
		return false
	}
	e := r.Decl.GetEnumeration()
	return e == nil || emit.Fielded(e)
}

// describe names a shape for an error message.
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

// comment writes a node's documentation and deprecation reason.
func comment(b *strings.Builder, indent string, meta *ir.Meta) {
	lines := emit.Doc(meta)
	if reason, ok := emit.Deprecated(meta); ok && reason != "" {
		if len(lines) > 0 {
			lines = append(lines, "")
		}
		lines = append(lines, "Deprecated: "+reason)
	}
	commentLines(b, indent, lines)
}

// commentLines writes lines as `//` comments, an empty line as a bare `//`.
func commentLines(b *strings.Builder, indent string, lines []string) {
	for _, line := range lines {
		if line == "" {
			fmt.Fprintf(b, "%s//\n", indent)
			continue
		}
		fmt.Fprintf(b, "%s// %s\n", indent, line)
	}
}

// options is a node's `name = value` options: `deprecated = true` when
// deprecated, then each `option` directive. An option naming deprecated is
// dropped on a deprecated node, since protoc refuses an option set twice.
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

// declOptions writes a declaration's options as statements, then its
// annotation when it has one.
func (g *generator) declOptions(b *strings.Builder, meta *ir.Meta, dirs []*ir.Directive, annotation string) {
	for _, o := range g.options(meta, dirs) {
		fmt.Fprintf(b, "  option %s;\n", o)
	}
	if annotation != "" {
		fmt.Fprintf(b, "  option %s;\n", annotation)
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

func validFile(name string) bool {
	return !strings.ContainsAny(name, `/\`) && strings.HasSuffix(name, ".proto")
}

// filePath places a file in the directories pkg spells, where buf expects
// it. An empty name stands for the package's last segment with .proto.
func filePath(pkg, name string) string {
	if name == "" {
		name = "model.proto"
		if pkg != "" {
			name = emit.LastSegment(pkg) + ".proto"
		}
	}
	return path.Join(strings.ReplaceAll(pkg, ".", "/"), name)
}
