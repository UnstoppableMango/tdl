// Package salesforce generates Salesforce DX source from a resolved model:
// a custom object for each entity, and an Apex class or enum for each value,
// mixin, and enum.
//
// docs/design/salesforce-backend.md has the mapping and the reasons for it.
// An entity is a row in the org, so it becomes a CustomObject with a
// CustomField per field it can store. Everything else is a shape Apex code
// passes around, so it becomes Apex, and Apex names an entity by its SObject
// type. The output is one file per component, which is how SFDX lays out a
// project, rather than the one file per model the schema backends write.
package salesforce

import (
	"context"
	"regexp"
	"slices"
	"strings"

	"github.com/unstoppablemango/tdl/backend/internal/emit"
	"github.com/unstoppablemango/tdl/ir"
	"github.com/unstoppablemango/tdl/plugin"
)

// Name is what this backend is called, in a target block and as
// tdl-gen-salesforce on PATH.
const Name = "salesforce"

// defaultAPIVersion is the Metadata API version an Apex class declares when
// the target block names none.
const defaultAPIVersion = "66.0"

// Backend implements [plugin.Backend].
type Backend struct{}

func (Backend) Describe() plugin.Description {
	str := []ir.LiteralKind{ir.LiteralKind_LITERAL_KIND_STRING}
	num := []ir.LiteralKind{ir.LiteralKind_LITERAL_KIND_INT}
	return plugin.Description{
		Name:    Name,
		Version: "0.1.0",
		// Each request is answered from the request alone.
		Reuse: true,
		Directives: []*plugin.DirectiveSpec{
			// The API name of an object, a field, or an Apex type, before
			// any __c suffix.
			{Name: "name", MinArgs: 1, MaxArgs: 1, ArgKinds: str},
			// The label an object or a field shows in the org.
			{Name: "label", MinArgs: 1, MaxArgs: 1, ArgKinds: str},
			// An object's plural label.
			{Name: "plural", MinArgs: 1, MaxArgs: 1, ArgKinds: str},
			// The field identifying an entity, which becomes a unique
			// external ID. The same directive as the go target's, so a
			// composite key is accepted here and warned about.
			{Name: "key", MinArgs: 1, MaxArgs: -1},
			// The length of a Text field.
			{Name: "length", MinArgs: 1, MaxArgs: 1, ArgKinds: num},
			// The decimal places of a Number field holding a decimal.
			{Name: "scale", MinArgs: 1, MaxArgs: 1, ArgKinds: num},
			// On the target block: prepended to every Apex type's name,
			// since an org has one namespace for classes.
			{Name: "prefix", MinArgs: 1, MaxArgs: 1, ArgKinds: str},
			// On the target block: the API version each Apex class declares.
			{Name: "apiVersion", MinArgs: 1, MaxArgs: 1, ArgKinds: str},
		},
	}
}

type generator struct {
	*emit.Session

	prefix     string
	apiVersion string

	// objects and classes are every name declared, lower case since
	// Salesforce compares names without case, with the declaration that
	// declared it.
	objects map[string]string
	classes map[string]string
}

// Generate returns the objects and Apex classes the model's own
// declarations become.
func (Backend) Generate(_ context.Context, req *plugin.Request) (*plugin.Response, error) {
	g := &generator{
		Session:    emit.NewSession(req, "Salesforce"),
		apiVersion: defaultAPIVersion,
		objects:    map[string]string{},
		classes:    map[string]string{},
	}
	if d, ok := g.Block("prefix"); ok {
		g.prefix = d.GetArgs()[0].GetText()
	}
	if d, ok := g.Block("apiVersion"); ok {
		g.apiVersion = d.GetArgs()[0].GetText()
	}

	own := g.Own()
	out := map[*ir.Decl][]*plugin.File{}
	skipped := map[*ir.Decl]bool{}
	for _, d := range own {
		files, err := g.decl(d)
		if err != nil {
			g.Warn(err)
			skipped[d] = true
			continue
		}
		out[d] = files
	}
	g.Cascade(own, skipped)

	var files []*plugin.File
	for _, d := range own {
		if !skipped[d] {
			files = append(files, out[d]...)
		}
	}
	return g.Response(files), nil
}

// decl renders one declaration into the files it becomes, which is none for
// one that declares nothing.
func (g *generator) decl(d *ir.Decl) ([]*plugin.File, error) {
	pos := d.GetMeta().GetPosition()
	name := d.GetMeta().GetName()

	switch {
	case d.GetClass() != nil:
		return nil, emit.Unsupported(pos, "%s is a class, and classes are not generated yet", name)
	case d.GetUnit() != nil:
		return nil, emit.Unsupported(pos, "%s is a unit, and units are not generated yet", name)
	case d.GetStructure() == nil && d.GetEnumeration() == nil && d.GetNewtype() == nil:
		// An alias is expanded where it is used, and a model's own
		// primitive names an opaque root; neither declares anything.
		return nil, nil
	}
	if len(d.Params()) > 0 {
		return nil, emit.Unsupported(pos, "%s is parameterized, and generics are not generated yet", name)
	}

	var files []*plugin.File
	var err error
	switch e := d.GetEnumeration(); {
	case d.GetNewtype() != nil:
		// Neither half has a distinct type to give a newtype, so it is
		// expanded where it is used. Resolving it here is what lets a
		// newtype over something unsupported be skipped, and cascade.
		var ref *emit.Ref
		if ref, err = g.Resolve(d.GetNewtype().GetBase()); err == nil {
			_, err = g.Expand(ref)
		}
	case d.GetStructure().GetKind() == ir.StructKind_STRUCT_KIND_ENTITY:
		files, err = g.object(d)
	case d.GetStructure() != nil:
		files, err = g.class(d)
	case emit.Fielded(e):
		files, err = g.sum(d)
	default:
		files, err = g.enum(d)
	}
	if err != nil {
		return nil, err
	}
	g.WarnConstraints(d)
	return files, nil
}

// apiName matches what Salesforce accepts as the name of an object, a
// field, or an Apex type: a letter, then letters, digits, and single
// underscores, not ending in one.
var apiName = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9]*(_[A-Za-z0-9]+)*$`)

// maxName is the longest name Salesforce accepts for an object, a field, or
// an Apex class, not counting a __c suffix.
const maxName = 40

// checkName says why a name cannot be an API name, if it cannot.
func checkName(pos *ir.Position, what, n string) error {
	switch {
	case !apiName.MatchString(n):
		return emit.Unsupported(pos, "%s would be named %q in Salesforce, which is not an API name", what, n)
	case len(n) > maxName:
		return emit.Unsupported(pos, "%s would be named %s in Salesforce, which is longer than %d characters", what, n, maxName)
	}
	return nil
}

// claim records that decl declares n, or says what already does.
func claim(names map[string]string, pos *ir.Position, decl, n string) error {
	key := strings.ToLower(n)
	if other, ok := names[key]; ok {
		return emit.Unsupported(pos, "%s would declare %s in Salesforce, and %s already does", decl, n, other)
	}
	names[key] = decl
	return nil
}

// label is a name as the org shows it: its words, capitalized, with spaces.
func label(name string) string {
	words := emit.Words(name)
	for i, w := range words {
		words[i] = strings.ToUpper(w[:1]) + w[1:]
	}
	return strings.Join(words, " ")
}

// description is a node's documentation and its deprecation, as one
// description.
func description(meta *ir.Meta) string {
	lines := slices.Clone(emit.Doc(meta))
	if reason, ok := emit.Deprecated(meta); ok {
		lines = append(lines, strings.TrimSpace("Deprecated. "+reason))
	}
	return strings.Join(lines, "\n")
}
