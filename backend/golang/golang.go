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
// Two shapes are worth knowing before reading the output. An enum whose
// variants carry no fields is a named string type with constants, and one
// where any variant carries fields is a sealed interface with a struct per
// variant; the language calls the second its sum type, and no single Go
// shape serves both. And three primitives, decimal, uuid, and date, map to
// a placeholder rather than a dependency this backend would be choosing on
// every consumer's behalf.
package golang

import (
	"context"
	"errors"
	"fmt"
	"go/format"
	"go/token"
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
		},
	}
}

// generator carries what rendering one request needs. A fresh one is made
// per request, so a reused connection shares nothing between them.
type generator struct {
	model  *ir.Model
	target string
	diags  []*plugin.Diagnostic

	// needTime is reset before each file, since imports are per file.
	needTime bool
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

	var files []*plugin.File
	for _, decl := range g.own() {
		file, err := g.file(pkg, decl)
		if err != nil {
			g.warn(err)
			continue
		}
		if file != nil {
			files = append(files, file)
		}
	}

	return &plugin.Response{Files: files, Diagnostics: g.diags}, nil
}

// file renders one declaration, or nil for one that generates nothing.
func (g *generator) file(pkg string, decl *ir.Decl) (*plugin.File, error) {
	g.needTime = false

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
		return nil, unsupported(decl.GetMeta().GetPosition(),
			"%s is a class, and classes are not generated yet", decl.GetMeta().GetName())
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
	if g.needTime {
		b.WriteString("\nimport \"time\"\n")
	}
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
// entity and a value are one shape apart in documentation only. Turning
// `key` into something is phase 2.
func (g *generator) structure(b *strings.Builder, decl *ir.Decl) error {
	if len(decl.Params()) > 0 {
		return unsupported(decl.GetMeta().GetPosition(),
			"%s is parameterized, and generics are not generated yet", decl.GetMeta().GetName())
	}

	name := g.declName(decl)
	g.doc(b, decl.GetMeta())
	fmt.Fprintf(b, "type %s struct {\n", name)
	if err := g.fields(b, decl.Fields()); err != nil {
		return err
	}
	b.WriteString("}\n")
	return nil
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
	if len(decl.Params()) > 0 {
		return unsupported(decl.GetMeta().GetPosition(),
			"%s is parameterized, and generics are not generated yet", decl.GetMeta().GetName())
	}

	name := g.declName(decl)
	variants := decl.GetEnumeration().GetVariants()

	carries := false
	for _, v := range variants {
		if len(v.GetFields()) > 0 {
			carries = true
			break
		}
	}

	if !carries {
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
		return nil
	}

	// A sealed interface: the unexported method is what keeps the set
	// closed, which is what makes this an enum rather than an open
	// hierarchy.
	sealed := "is" + name
	g.doc(b, decl.GetMeta())
	fmt.Fprintf(b, "type %s interface{ %s() }\n", name, sealed)

	for _, v := range variants {
		variant := name + exported(v.GetMeta().GetName())
		b.WriteString("\n")
		g.doc(b, v.GetMeta())
		fmt.Fprintf(b, "type %s struct {\n", variant)
		if err := g.fields(b, v.GetFields()); err != nil {
			return err
		}
		b.WriteString("}\n\n")
		fmt.Fprintf(b, "func (%s) %s() {}\n", variant, sealed)
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
	if len(decl.Params()) > 0 {
		return unsupported(decl.GetMeta().GetPosition(),
			"%s is parameterized, and generics are not generated yet", decl.GetMeta().GetName())
	}

	base, err := g.goType(decl.GetNewtype().GetBase())
	if err != nil {
		return err
	}

	if n := len(decl.GetNewtype().GetValueConstraints()); n > 0 {
		g.warn(unsupported(decl.GetMeta().GetPosition(),
			"%s carries %d where constraint(s), and validation is not generated yet",
			decl.GetMeta().GetName(), n))
	}

	g.doc(b, decl.GetMeta())
	fmt.Fprintf(b, "type %s %s\n", g.declName(decl), base)
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
	d := &plugin.Diagnostic{
		Severity: plugin.Severity_SEVERITY_WARNING,
		Message:  err.Error(),
	}
	var u *unsupportedError
	if errors.As(err, &u) {
		d.Position = u.position
	}
	g.diags = append(g.diags, d)
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
		if d.GetMeta().GetPosition().GetFilename() == prelude.Name {
			continue
		}
		own = append(own, d)
	}
	return own
}
