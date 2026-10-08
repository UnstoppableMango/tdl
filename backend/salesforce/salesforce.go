// Package salesforce generates Salesforce DX source, one file per
// component, from a resolved model: a custom object for each entity, and an
// Apex class or enum for each value, mixin, and enum. The mapping is in
// docs/design/salesforce-backend.md.
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

// Name names this backend in a target block and as tdl-gen-salesforce.
const Name = "salesforce"

const defaultAPIVersion = "66.0"

// Backend implements [plugin.Backend].
type Backend struct{}

func (Backend) Describe() plugin.Description {
	str := []ir.LiteralKind{ir.LiteralKind_LITERAL_KIND_STRING}
	num := []ir.LiteralKind{ir.LiteralKind_LITERAL_KIND_INT}
	return plugin.Description{
		Name:    Name,
		Version: "0.1.0",
		// Each request stands alone.
		Reuse: true,
		Directives: []*plugin.DirectiveSpec{
			// The API name of an object, field, or Apex type, before any
			// __c suffix.
			{Name: "name", MinArgs: 1, MaxArgs: 1, ArgKinds: str},
			// The label an object or a field shows in the org.
			{Name: "label", MinArgs: 1, MaxArgs: 1, ArgKinds: str},
			// An object's plural label.
			{Name: "plural", MinArgs: 1, MaxArgs: 1, ArgKinds: str},
			// The field identifying an entity, made a unique external ID.
			// It matches the go target's, so a composite key is accepted
			// and warned about.
			{Name: "key", MinArgs: 1, MaxArgs: -1},
			// The length of a Text field.
			{Name: "length", MinArgs: 1, MaxArgs: 1, ArgKinds: num},
			// The decimal places of a Number field holding a decimal.
			{Name: "scale", MinArgs: 1, MaxArgs: 1, ArgKinds: num},
			// On the target block: prepended to every Apex type's name,
			// since an org has one class namespace.
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

	// objects and classes map each declared name, lower-cased since
	// Salesforce ignores case, to the declaration declaring it.
	objects map[string]string
	classes map[string]string
}

// Generate returns the objects and Apex classes the model's declarations
// become.
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

// decl renders one declaration into its files, if any.
func (g *generator) decl(d *ir.Decl) ([]*plugin.File, error) {
	if ok, err := emit.Declares(d); !ok {
		return nil, err
	}

	var files []*plugin.File
	var err error
	switch e := d.GetEnumeration(); {
	case d.GetNewtype() != nil:
		// A newtype is expanded where used. Resolving it here lets an
		// unsupported one be skipped and cascade.
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

// apiName matches a Salesforce API name: a letter, then letters, digits,
// and single underscores, not ending in one.
var apiName = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9]*(_[A-Za-z0-9]+)*$`)

// maxName is the longest API name Salesforce accepts, not counting __c.
const maxName = 40

// checkName says why n cannot be an API name, if it cannot.
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

// label is a name as the org shows it: capitalized words with spaces.
func label(name string) string {
	words := emit.Words(name)
	for i, w := range words {
		words[i] = strings.ToUpper(w[:1]) + w[1:]
	}
	return strings.Join(words, " ")
}

// description is a node's documentation and deprecation.
func description(meta *ir.Meta) string {
	lines := slices.Clone(emit.Doc(meta))
	if reason, ok := emit.Deprecated(meta); ok {
		lines = append(lines, strings.TrimSpace("Deprecated. "+reason))
	}
	return strings.Join(lines, "\n")
}
