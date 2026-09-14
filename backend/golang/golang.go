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
	"fmt"
	"go/format"
	"go/token"
	"slices"
	"strconv"
	"strings"

	"github.com/unstoppablemango/tdl/backend/internal/emit"
	"github.com/unstoppablemango/tdl/ir"
	"github.com/unstoppablemango/tdl/plugin"
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
	*emit.Session

	// cur is the index of the declaration being rendered, whose type
	// parameters a bare parameter reference names.
	cur int32

	// needsComparable holds, per parameterized declaration, which of its
	// parameters Go needs to be comparable.
	needsComparable map[int32][]bool

	// genClass holds the classes generated as interfaces, and marks the
	// classes whose marker each declaration carries, both by index.
	genClass map[int32]bool
	// classCycle holds the classes whose requires clause reaches themselves,
	// which a Go interface cannot embed its way out of.
	classCycle map[int32]bool
	marks      map[int32][]int32

	// curClasses is the generated classes constraining each parameter of the
	// declaration being rendered.
	curClasses [][]int32

	// valid holds what has something to check and carries Validate.
	valid map[validKey]bool

	// imports holds the packages the file being rendered uses, by path. It
	// is reset before each file, since imports are per file.
	imports map[string]bool
}

// Generate returns one Go file per declaration the model owns.
func (Backend) Generate(_ context.Context, req *plugin.Request) (*plugin.Response, error) {
	g := &generator{Session: emit.NewSession(req, "Go")}

	pkg := packageClause(g.Model.GetPackage())
	var pkgPos *ir.Position
	if d, ok := g.Block("package"); ok {
		pkg, pkgPos = packageClause(d.GetArgs()[0].GetText()), d.GetPosition()
	}

	// A package clause is one identifier and every file carries it, so a
	// name Go will not accept is not a declaration to skip but the whole
	// output. It is an error rather than a warning for that reason: the
	// host writes nothing and says why.
	if token.IsKeyword(pkg) {
		g.Error(pkgPos, "%q is a Go keyword and cannot be a package name", pkg)
		return g.Response(nil), nil
	}

	// Which classes are generated, which declarations carry each marker, and
	// which parameters Go needs to be comparable are decided before anything
	// is rendered, since a use of a constrained declaration anywhere in the
	// table asks about them.
	g.planClasses()
	g.inferComparable()
	g.planValidation()

	skipped := map[*ir.Decl]bool{}
	rendered := map[*ir.Decl]*plugin.File{}
	for i, decl := range g.Model.GetDecls() {
		if !emit.IsOwn(decl) {
			continue
		}
		g.cur = int32(i)
		file, err := g.file(pkg, decl)
		if err != nil {
			g.Warn(err)
			skipped[decl] = true
			continue
		}
		if file != nil {
			rendered[decl] = file
		}
	}

	// A declaration naming a skipped one would name a type the package does
	// not declare, so it is skipped too, and its file is dropped rather than
	// written.
	own := g.Own()
	g.Cascade(own, skipped)

	var files []*plugin.File
	for _, decl := range own {
		if file := rendered[decl]; file != nil && !skipped[decl] {
			files = append(files, file)
		}
	}

	return g.Response(files), nil
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
		return nil, emit.Unsupported(decl.GetMeta().GetPosition(),
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
		g.Error(decl.GetMeta().GetPosition(), "%s is not parseable Go: %v", path, err)
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

	if d, ok := g.Find(decl.GetDirectives(), "key"); ok {
		if err := g.key(b, decl, name, d); err != nil {
			g.Warn(err)
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

	if !emit.Fielded(decl.GetEnumeration()) {
		// Turning the enum into the sealed shape to keep its parameters
		// would be a second rule deciding the shape, and nothing in it names
		// a parameter, since nothing carries a field.
		if len(decl.Params()) > 0 {
			g.Warn(emit.Unsupported(decl.GetMeta().GetPosition(),
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
		return emit.Unsupported(decl.GetMeta().GetPosition(),
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
	lines := emit.Doc(meta)
	for _, line := range lines {
		fmt.Fprintf(b, "// %s\n", line)
	}
	if reason, ok := emit.Deprecated(meta); ok {
		if len(lines) > 0 {
			b.WriteString("//\n")
		}
		if reason == "" {
			reason = "this declaration is on its way out."
		}
		fmt.Fprintf(b, "// Deprecated: %s\n", reason)
	}
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

// declName is the Go identifier for a declaration, after a `name`
// directive has had its say.
func (g *generator) declName(decl *ir.Decl) string {
	return g.DeclName(decl, exported)
}

// fieldName is the Go identifier for a field.
func (g *generator) fieldName(f *ir.Field) string {
	return g.FieldName(f, exported)
}
