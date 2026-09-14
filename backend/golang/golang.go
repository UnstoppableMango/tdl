// Package golang generates Go source from a resolved model.
//
// It is the first code generator in this repository, and the first thing to
// answer the question the plugin protocol deliberately left open: what
// generated code should look like. See docs/design/go-backend.md for the
// decisions and docs/design/go-backend-plan.md for what each phase adds.
//
// The package is called golang and the backend is called "go", because a
// target block writes the latter and a Go package cannot usefully be the
// former.
//
// Three things are worth knowing before reading the output. An enum whose
// variants carry no fields is a named string type with constants, and one
// where any variant carries fields is a sealed interface with a struct per
// variant; the language calls the second its sum type, and no single Go
// shape serves both. Three primitives, decimal, uuid, and date, map to a
// placeholder rather than a dependency this backend would be choosing on
// every consumer's behalf. And a class is an interface with one unexported
// method, which each declaration satisfying the class carries, so a
// `requires` clause is a Go constraint and conformance stays declared.
package golang

import (
	"context"
	"errors"
	"fmt"
	"go/format"
	"go/token"
	"slices"
	"strconv"
	"strings"

	"github.com/unstoppablemango/tdl/ir"
	"github.com/unstoppablemango/tdl/plugin"
	"github.com/unstoppablemango/tdl/prelude"
)

// Name is what this backend is called, in a target block and as tdl-gen-go
// on PATH.
const Name = "go"

// Backend implements [plugin.Backend].
type Backend struct{}

func (Backend) Describe() plugin.Description {
	str := []ir.LiteralKind{ir.LiteralKind_LITERAL_KIND_STRING}
	return plugin.Description{
		Name:    Name,
		Version: "0.1.0",
		// Each request is answered from the request alone, so serving
		// several on one connection is safe.
		Reuse: true,
		Directives: []*plugin.DirectiveSpec{
			// The Go package clause, written as an import path because that
			// is what a consumer of the generated code writes.
			{Name: "package", MinArgs: 1, MaxArgs: 1, ArgKinds: str},
			// The Go identifier for a declaration or a field, when the TDL
			// name is not the one wanted.
			{Name: "name", MinArgs: 1, MaxArgs: 1, ArgKinds: str},
			// A struct tag, emitted verbatim. A tag is an open convention,
			// so this backend does not parse it.
			{Name: "tag", MinArgs: 1, MaxArgs: 1, ArgKinds: str},
			// The fields identifying an entity, as bare names. ArgKinds
			// constrains by position, so that each one is a name is checked
			// here rather than declared.
			{Name: "key", MinArgs: 1, MaxArgs: -1},
		},
	}
}

// generator carries what rendering one request needs. A fresh one is made
// per request, so a reused connection shares nothing between them.
type generator struct {
	model  *ir.Model
	target string
	diags  []*plugin.Diagnostic

	// skipped holds the declarations not generated, by index, with why.
	skipped map[int32]error

	// cur is the index of the declaration being rendered, whose type
	// parameters a bare parameter reference names.
	cur int32

	// needsComparable holds, per parameterized declaration, which of its
	// parameters Go needs to be comparable.
	needsComparable map[int32][]bool

	// genClass holds the classes generated as interfaces, and marks the
	// classes whose marker each declaration carries, both by index.
	genClass map[int32]bool
	marks    map[int32][]int32

	// curClasses is the generated classes constraining each parameter of the
	// declaration being rendered.
	curClasses [][]int32

	// planned holds what deciding the classes found to warn about, which
	// belongs to no single render.
	planned []*plugin.Diagnostic

	// valid holds what has something to check and carries Validate.
	valid map[validKey]bool

	// imports holds the packages the file being rendered uses, by path. It
	// is reset before each file, since imports are per file.
	imports map[string]bool
}

// Generate returns one Go file per declaration the model owns.
func (Backend) Generate(_ context.Context, req *plugin.Request) (*plugin.Response, error) {
	g := &generator{model: req.GetModel(), target: req.GetTarget()}

	pkg := packageClause(g.model.GetPackage())
	var pkgPos *ir.Position
	if d, ok := g.blockDirective("package"); ok {
		pkg, pkgPos = packageClause(d.GetArgs()[0].GetText()), d.GetPosition()
	}

	// A package clause is one identifier and every file carries it, so a
	// name Go will not accept is not a declaration to skip but the whole
	// output. It is an error rather than a warning for that reason: the
	// host writes nothing and says why.
	if token.IsKeyword(pkg) {
		g.diags = append(g.diags, &plugin.Diagnostic{
			Severity: plugin.Severity_SEVERITY_ERROR,
			Message:  fmt.Sprintf("%q is a Go keyword and cannot be a package name", pkg),
			Position: pkgPos,
		})
		return &plugin.Response{Diagnostics: g.diags}, nil
	}

	// A declaration naming a skipped one would name a type the package does
	// not declare, so it is skipped too. It may come earlier in the table
	// than what it names, so rendering repeats until a pass skips nothing
	// new. The skipped set only grows, which bounds the passes by the number
	// of declarations.
	g.skipped = map[int32]error{}
	g.planClasses()
	g.inferComparable()
	g.planValidation()
	var files []*plugin.File
	var rendered map[int32][]*plugin.Diagnostic
	for again := true; again; {
		again = false
		files, rendered = nil, map[int32][]*plugin.Diagnostic{}
		for i, decl := range g.model.GetDecls() {
			index := int32(i)
			if !isOwn(decl) || g.skipped[index] != nil {
				continue
			}
			g.diags, g.cur = nil, index
			file, err := g.file(pkg, decl)
			if err != nil {
				g.skipped[index] = err
				again = true
				continue
			}
			rendered[index] = g.diags
			if file != nil {
				files = append(files, file)
			}
		}
	}

	// Diagnostics come from the last pass that rendered each declaration, so
	// a repeated pass does not repeat them, and in declaration order.
	g.diags = nil
	for i := range g.model.GetDecls() {
		index := int32(i)
		if err := g.skipped[index]; err != nil {
			g.warn(err)
		}
		g.diags = append(g.diags, rendered[index]...)
	}
	g.diags = append(g.diags, g.planned...)

	return &plugin.Response{Files: files, Diagnostics: g.diags}, nil
}

// file renders one declaration, or nil for one that generates nothing.
func (g *generator) file(pkg string, decl *ir.Decl) (*plugin.File, error) {
	g.imports = map[string]bool{}
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
		// An alias is transparent and is expanded at every use, so there is
		// nothing to declare for it.
		return nil, nil
	case decl.GetClass() != nil:
		if err := g.class(&body, decl); err != nil {
			return nil, err
		}
	case decl.GetUnit() != nil:
		return nil, unsupported(decl.GetMeta().GetPosition(),
			"%s is a unit, and units are not generated yet", decl.GetMeta().GetName())
	case decl.GetPrimitive() != nil:
		// A model declaring its own primitive is naming an opaque root
		// type, which has no Go declaration to make.
		return nil, nil
	default:
		return nil, nil
	}

	var b strings.Builder
	b.WriteString("// Code generated by tdl. DO NOT EDIT.\n\n")
	fmt.Fprintf(&b, "package %s\n", pkg)
	writeImports(&b, g.imports)
	b.WriteString("\n")
	b.WriteString(body.String())

	path := fileName(decl.GetMeta().GetName())
	src, err := format.Source([]byte(b.String()))
	if err != nil {
		// Malformed output the consumer can read beats no output, and a
		// generator that cannot produce parseable Go has a bug that should
		// be visible rather than swallowed.
		g.diags = append(g.diags, &plugin.Diagnostic{
			Severity: plugin.Severity_SEVERITY_ERROR,
			Message:  fmt.Sprintf("%s is not parseable Go: %v", path, err),
			Position: decl.GetMeta().GetPosition(),
		})
		src = []byte(b.String())
	}

	return &plugin.File{Path: path, Content: src}, nil
}

// structure renders an entity, a value, or a mixin.
//
// The three differ in what they mean rather than in what they emit: Go has
// no way to say "identity that survives changes to its contents", so an
// entity and a value are one shape apart until a `key` directive names the
// fields identifying the entity.
func (g *generator) structure(b *strings.Builder, decl *ir.Decl) error {
	if err := g.paramProblem(decl); err != nil {
		return err
	}

	name := g.declName(decl)
	g.doc(b, decl.GetMeta())
	fmt.Fprintf(b, "type %s%s struct {\n", name, g.typeParams(decl))
	if err := g.fields(b, decl.Fields()); err != nil {
		return err
	}
	b.WriteString("}\n")

	if d, ok := g.find(decl.GetDirectives(), "key"); ok {
		if err := g.key(b, decl, name, d); err != nil {
			g.warn(err)
		}
	}
	g.writeValidation(b, decl, -1)
	g.writeMarkers(b, name+typeArgs(decl))
	return nil
}

// key renders an entity's identity: a Key method returning the one field a
// `key` directive names, or a key type holding every field it names.
//
// A key that cannot be generated is returned as an error for the caller to
// warn about, and the entity is still emitted, since the struct is what
// other declarations name.
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

	// A generic entity's key type takes the entity's parameters, since its
	// fields may name them.
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
		return nil, unsupported(pos, "%s is named \"\" in this target, and a Key method needs a receiver named after it", decl.GetMeta().GetName())
	}
	if decl.GetStructure().GetKind() != ir.StructKind_STRUCT_KIND_ENTITY {
		return nil, unsupported(pos, "%s has a key, and only an entity is identified by one", name)
	}
	for _, f := range decl.Fields() {
		if g.fieldName(f) == "Key" {
			return nil, unsupported(pos, "%s has a field named Key, which the Key method would collide with", name)
		}
	}

	var fields []*ir.Field
	for _, arg := range d.GetArgs() {
		if arg.GetKind() != ir.LiteralKind_LITERAL_KIND_NAME {
			return nil, unsupported(at(arg), "a key names fields, and %q is %s", arg.GetText(), ir.KindName(arg.GetKind()))
		}
		i := slices.IndexFunc(decl.Fields(), func(f *ir.Field) bool { return f.GetMeta().GetName() == arg.GetText() })
		if i < 0 {
			return nil, unsupported(at(arg), "%s has no field %s to key on", name, arg.GetText())
		}
		f := decl.Fields()[i]
		if slices.Contains(fields, f) {
			return nil, unsupported(at(arg), "the key of %s names %s twice", name, arg.GetText())
		}
		if !g.comparable(f.GetType()) {
			return nil, unsupported(at(arg), "a key is compared, and %s.%s is not a comparable Go type", name, arg.GetText())
		}
		fields = append(fields, f)
	}

	if len(fields) > 1 {
		keyType := name + "Key"
		for _, other := range g.own() {
			if other != decl && g.declName(other) == keyType {
				return nil, unsupported(pos, "the key type %s would collide with the declaration of that name", keyType)
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
		if tag, ok := g.directive(f.GetDirectives(), "tag"); ok {
			fmt.Fprintf(b, " %s", quoteTag(tag))
		}
		b.WriteString("\n")
	}
	return nil
}

// quoteTag writes a struct tag as a raw string, which is how every struct
// tag in Go is written, and falls back to an interpreted string for the one
// that cannot be: a tag containing a backquote.
func quoteTag(tag string) string {
	if !strings.Contains(tag, "`") {
		return "`" + tag + "`"
	}
	return strconv.Quote(tag)
}

// enumeration renders an enum as one of two shapes, chosen by whether any
// variant carries fields.
//
// docs/design/go-backend.md argues the split. In short: an interface for a
// three-name enum is unusable as a map key and unwritable as a constant,
// and constants cannot express a variant with fields at all.
func (g *generator) enumeration(b *strings.Builder, decl *ir.Decl) error {
	name := g.declName(decl)
	variants := decl.GetEnumeration().GetVariants()

	if !carries(decl.GetEnumeration()) {
		// Turning the enum into the sealed shape to keep its parameters
		// would be a second rule deciding the shape, and nothing in it names
		// a parameter, since nothing carries a field.
		if len(decl.Params()) > 0 {
			g.warn(unsupported(decl.GetMeta().GetPosition(),
				"%s takes type parameters and none of its variants carries a field, so it is constants, and a Go constant cannot be generic: its parameters are dropped",
				decl.GetMeta().GetName()))
		}
		g.doc(b, decl.GetMeta())
		fmt.Fprintf(b, "type %s string\n\nconst (\n", name)
		for _, v := range variants {
			// The string is the variant's name as written. Inventing a wire
			// format would be deciding something that belongs to the
			// consumer, and `tag` is where they decide it.
			fmt.Fprintf(b, "\t%s%s %s = %q\n",
				name, exported(v.GetMeta().GetName()), name, v.GetMeta().GetName())
		}
		b.WriteString(")\n")
		g.writeMarkers(b, name)
		return nil
	}

	// A sealed interface: the unexported method is what keeps the set
	// closed, which is what makes this an enum rather than an open
	// hierarchy.
	if err := g.paramProblem(decl); err != nil {
		return err
	}

	// A generic enum's marker takes its parameters, so a variant of one
	// instantiation does not satisfy another: without them a ResultOk[int]
	// would be a Result[string].
	sealed := "is" + name + "(" + strings.Join(paramNames(decl), ", ") + ")"
	params, args := g.typeParams(decl), typeArgs(decl)
	g.doc(b, decl.GetMeta())
	// An interface cannot carry a class's marker, so it embeds the class,
	// and every variant carries the marker instead.
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
		variant := name + exported(v.GetMeta().GetName())
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
//
// Its `where` constraints are not enforced. Validation is phase 4 and its
// own set of decisions about where a check lives.
//
// What is skipped is the constraint and not the declaration. Emitting
// nothing for a constrained newtype would leave every field naming it
// referring to a type the package does not declare, so the type is emitted
// and the unenforced constraint is said out loud.
func (g *generator) newtype(b *strings.Builder, decl *ir.Decl) error {
	if err := g.paramProblem(decl); err != nil {
		return err
	}

	base, err := g.goType(decl.GetNewtype().GetBase())
	if err != nil {
		return err
	}
	// Go refuses `type N[T any] T`: a type parameter cannot be the whole of
	// a type declaration.
	if slices.Contains(paramNames(decl), base) {
		return unsupported(decl.GetMeta().GetPosition(),
			"%s is a newtype over its type parameter %s, and Go cannot declare a type that is only a type parameter",
			decl.GetMeta().GetName(), base)
	}

	g.doc(b, decl.GetMeta())
	fmt.Fprintf(b, "type %s%s %s\n", g.declName(decl), g.typeParams(decl), base)
	g.writeValidation(b, decl, -1)
	g.writeMarkers(b, g.declName(decl)+typeArgs(decl))
	return nil
}

// doc writes a node's documentation as Go doc comments, and its deprecation
// as the paragraph go doc and every editor reads.
func (g *generator) doc(b *strings.Builder, meta *ir.Meta) {
	for _, line := range meta.GetDoc() {
		fmt.Fprintf(b, "// %s\n", strings.TrimSpace(line))
	}
	if d := meta.GetDeprecated(); d != nil {
		if len(meta.GetDoc()) > 0 {
			b.WriteString("//\n")
		}
		reason := d.GetReason()
		if reason == "" {
			reason = "this declaration is on its way out."
		}
		fmt.Fprintf(b, "// Deprecated: %s\n", reason)
	}
}

// declName is the Go identifier for a declaration, after a `name`
// directive has had its say.
func (g *generator) declName(decl *ir.Decl) string {
	if s, ok := g.directive(decl.GetDirectives(), "name"); ok {
		return s
	}
	return exported(decl.GetMeta().GetName())
}

// fieldName is the Go identifier for a field.
func (g *generator) fieldName(f *ir.Field) string {
	if s, ok := g.directive(f.GetDirectives(), "name"); ok {
		return s
	}
	return exported(f.GetMeta().GetName())
}

// directive returns the single string argument of a directive on a node.
func (g *generator) directive(all []*ir.Directive, name string) (string, bool) {
	if d, ok := g.find(all, name); ok {
		return d.GetArgs()[0].GetText(), true
	}
	return "", false
}

// find returns a directive carrying at least one argument.
//
// A model carries directives for every target block in it, tagged with the
// block they came from, so this filters rather than assuming what it is
// handed is its own.
func (g *generator) find(all []*ir.Directive, name string) (*ir.Directive, bool) {
	for _, d := range plugin.Directives(g.target, all) {
		if d.GetName() != name {
			continue
		}
		if len(d.GetArgs()) > 0 {
			return d, true
		}
	}
	return nil, false
}

// blockDirective reads a directive written on the target block itself
// rather than against a node in the model.
//
// It returns the directive rather than its argument, because a diagnostic
// about the value wants the position it was written at.
func (g *generator) blockDirective(name string) (*ir.Directive, bool) {
	for _, block := range g.model.GetTargets() {
		if block.GetMeta().GetName() != g.target {
			continue
		}
		if d, ok := g.find(block.GetDirectives(), name); ok {
			return d, true
		}
	}
	return nil, false
}

// warn reports something the backend cannot handle, with a position when
// the failure carried one.
//
// A backend says what it cannot do here rather than returning an error,
// because this reaches the user with a position attached and does not stop
// the run.
func (g *generator) warn(err error) {
	g.diags = append(g.diags, warning(err))
}

// use records that the file being rendered imports a package.
func (g *generator) use(path string) {
	g.imports[path] = true
}

// importNames is every package name generated code may import, by the name
// it is referred to with.
var importNames = map[string]bool{"errors": true, "fmt": true, "regexp": true, "time": true, "utf8": true}

// writeImports writes a file's import declaration: one import on its own
// line, several as a sorted block.
func writeImports(b *strings.Builder, imports map[string]bool) {
	paths := make([]string, 0, len(imports))
	for p := range imports {
		paths = append(paths, p)
	}
	slices.Sort(paths)

	switch len(paths) {
	case 0:
	case 1:
		fmt.Fprintf(b, "\nimport %q\n", paths[0])
	default:
		b.WriteString("\nimport (\n")
		for _, p := range paths {
			fmt.Fprintf(b, "\t%q\n", p)
		}
		b.WriteString(")\n")
	}
}

// warning is the diagnostic [generator.warn] reports.
func warning(err error) *plugin.Diagnostic {
	d := &plugin.Diagnostic{
		Severity: plugin.Severity_SEVERITY_WARNING,
		Message:  err.Error(),
	}
	var u *unsupportedError
	if errors.As(err, &u) {
		d.Position = u.position
	}
	return d
}

// own returns the declarations the model's own file declared.
//
// The prelude is merged into the declaration table untagged, so a model
// whose source declares two things arrives with twenty-one declarations. A
// backend that emits per declaration has to decide what is the user's, and
// which file a declaration came from is what says so.
//
// The embedded prelude is named [prelude.Name] and nothing else is: it is
// parsed under that name rather than read from a path, so the comparison is
// against the whole name and not its ending. A user's `my-std.tdl`, or a
// `std.tdl` of their own in any directory, is theirs and is generated.
//
// A replacement prelude passed to `sema.WithPrelude` is named by whoever
// passed it and is not recognized here. Marking the prelude on the wire is
// the fix, and `plugins.md` argues the opposite, that a backend should see
// prelude declarations as declarations like any other; until that is
// settled, a project replacing the prelude generates it too.
func (g *generator) own() []*ir.Decl {
	var own []*ir.Decl
	for _, d := range g.model.GetDecls() {
		if isOwn(d) {
			own = append(own, d)
		}
	}
	return own
}

// isOwn reports whether a declaration is the model's rather than the
// prelude's; [generator.own] says how the two are told apart.
func isOwn(d *ir.Decl) bool {
	return d.GetMeta().GetPosition().GetFilename() != prelude.Name
}
