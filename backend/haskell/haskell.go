// Package haskell generates a Haskell module from a resolved model: a
// record for each struct, a sum type for each enum, a newtype for each
// newtype, and a synonym for each alias. The mapping is in
// docs/architecture/haskell-backend.md.
package haskell

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/unstoppablemango/tdl/backend/internal/emit"
	"github.com/unstoppablemango/tdl/ir"
	"github.com/unstoppablemango/tdl/plugin"
)

// Name names this backend in a target block and as tdl-gen-haskell.
const Name = "haskell"

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
			// The module name, and so the file's path, on the target block.
			{Name: "module", MinArgs: 1, MaxArgs: 1, ArgKinds: str, Placements: plugin.On(plugin.OnBlock)},
			// The Haskell identifier for a declaration, field, or variant.
			{Name: "name", MinArgs: 1, MaxArgs: 1, ArgKinds: str, Placements: plugin.OnMembers()},
			// Writes an enum's variants, or every enum's on the target block,
			// behind the enum's name.
			{Name: "prefix", Placements: plugin.On(plugin.OnBlock, plugin.OnDecl)},
			// A lazy field under StrictData.
			{Name: "lazy", Placements: plugin.On(plugin.OnField)},
			// The module and type a declaration maps to, declared elsewhere.
			{Name: "foreign", MinArgs: 2, MaxArgs: 2, ArgKinds: str, Placements: plugin.On(plugin.OnDecl)},
			// The fields identifying an entity.
			{Name: "key", MinArgs: 1, MaxArgs: -1, Placements: plugin.On(plugin.OnDecl)},
			// An anyclass instance of a class, from the module named.
			{Name: "derive", MinArgs: 2, MaxArgs: 2, ArgKinds: str, Repeatable: true, Placements: plugin.On(plugin.OnBlock, plugin.OnDecl)},
		},
	}
}

// pragmas are the extensions every module turns on. GHC2021 is written
// rather than left to the consumer's build, so the module means the same
// thing under any.
var pragmas = []string{
	"GHC2021",
	"DerivingStrategies",
	"DuplicateRecordFields",
	"NoFieldSelectors",
	"OverloadedRecordDot",
	"StrictData",
}

var (
	eq      = imported{"Prelude", "Eq"}
	ord     = imported{"Prelude", "Ord"}
	show    = imported{"Prelude", "Show"}
	read    = imported{"Prelude", "Read"}
	enum    = imported{"Prelude", "Enum"}
	bounded = imported{"Prelude", "Bounded"}
	generic = imported{"GHC.Generics", "Generic"}
)

type generator struct {
	*emit.Session

	// prefix is whether the target block prefixes every enum's variants.
	prefix bool

	// types and ctors map each Haskell type and constructor name to the
	// declaration declaring it, since each is one namespace across the
	// module.
	types map[string]string
	ctors map[string]string

	// uses holds every imported name, by module, and exts every extension
	// beyond pragmas, for the declaration being rendered.
	uses map[string]map[string]bool
	exts map[string]bool
	// needs holds every class and type a render names that another render
	// may fail to declare.
	needs []*ir.Decl
	// expand renders a synonym the model declares as what it names, for
	// telling instances apart.
	expand bool

	// self is the generated module.
	self string

	// foreign and externs map a declaration and an extern to the type
	// another module declares for it.
	foreign map[*ir.Decl]imported
	externs map[*ir.Extern]imported

	// names maps every type and constructor name the model's declarations
	// take to the declaration taking it, so a key record can tell it would
	// collide with one declared after it.
	names map[string]string

	// warned holds each directive already warned about.
	warned map[*ir.Directive]bool

	// functors holds each declaration deriving Functor, Foldable, and
	// Traversable.
	functors map[*ir.Decl]bool
}

// rendered is one declaration's output.
type rendered struct {
	text string

	// types is the type names it declares.
	types []string

	// labels is the field labels it declares, each once.
	labels []string

	// deprecated is a DEPRECATED pragma for each deprecated field, by its
	// label, written only when no other declaration shares the label.
	deprecated []fieldPragma

	// uses is what it imports, by module, and exts the extensions it
	// needs.
	uses  map[string]map[string]bool
	exts  map[string]bool
	needs []*ir.Decl
}

type fieldPragma struct{ label, pragma string }

// Generate returns one module holding the model's declarations.
func (Backend) Generate(_ context.Context, req *plugin.Request) (*plugin.Response, error) {
	g := &generator{
		Session: emit.NewSession(req, "Haskell"),
		types:   map[string]string{},
		ctors:   map[string]string{},
		names:   map[string]string{},
		warned:  map[*ir.Directive]bool{},
	}
	g.prefix = g.Bare("prefix")

	module, err := g.module()
	if err != nil {
		g.Warn(err)
		return g.Response(nil), nil
	}
	g.self = module
	g.planForeign()

	own := g.Own()
	g.planFunctors(own)
	for _, d := range own {
		g.claim(d)
	}
	// Read the target block's derive directives once, so a problem with
	// one warns whether or not a declaration is emitted.
	g.derived(&ir.Decl{})
	out := map[*ir.Decl]*rendered{}
	skipped := map[*ir.Decl]bool{}
	// Each declaration's warnings are held until the cascade decides
	// whether it is emitted.
	held := map[*ir.Decl][]*plugin.Diagnostic{}
	for _, d := range own {
		mark := len(g.Diags)
		g.begin()
		r, err := g.decl(d)
		if r != nil {
			r.uses, r.exts, r.needs = g.uses, g.exts, g.needs
		}
		if err != nil {
			g.Diags = g.Diags[:mark]
			g.Warn(err)
			skipped[d] = true
			continue
		}
		held[d] = slices.Clone(g.Diags[mark:])
		g.Diags = g.Diags[:mark]
		out[d] = r
	}
	g.Cascade(own, skipped)
	// Cascade passes over aliases, which other backends expand, and a
	// synonym naming a skipped declaration would not compile.
	for _, d := range own {
		if d.GetAlias() == nil || skipped[d] || out[d] == nil {
			continue
		}
		for _, ref := range g.References(d) {
			if skipped[ref] {
				skipped[d] = true
				g.Warn(emit.Unsupported(d.GetMeta().GetPosition(),
					"%s names %s, which is not generated", d.GetMeta().GetName(), ref.GetMeta().GetName()))
				break
			}
		}
	}
	// A class or a function constrained by one names classes as well as
	// types.
	for changed := true; changed; {
		changed = false
		for _, d := range own {
			if skipped[d] || out[d] == nil {
				continue
			}
			for _, n := range out[d].needs {
				if skipped[n] {
					skipped[d] = true
					changed = true
					g.Warn(emit.Unsupported(d.GetMeta().GetPosition(),
						"%s names %s, which is not generated", d.GetMeta().GetName(), n.GetMeta().GetName()))
					break
				}
			}
		}
		if changed {
			g.Cascade(own, skipped)
		}
	}
	instances := g.instances(own, skipped)
	after := map[*ir.Decl][]*rendered{}
	var trailing []*rendered
	for _, inst := range instances {
		if inst.after != nil {
			after[inst.after] = append(after[inst.after], inst.r)
		} else {
			trailing = append(trailing, inst.r)
		}
	}

	// A skipped declaration's imports would be unused, which -Wall refuses.
	g.uses = map[string]map[string]bool{}
	var emitted []*rendered
	declared := map[string]bool{}
	labels := map[string]int{}
	exts := map[string]bool{}
	for _, d := range own {
		if r := out[d]; r != nil && !skipped[d] {
			g.Diags = append(g.Diags, held[d]...)
			emitted = append(emitted, r)
			emitted = append(emitted, after[d]...)
		}
	}
	emitted = append(emitted, trailing...)
	for _, r := range emitted {
		for _, t := range r.types {
			declared[t] = true
		}
		for _, l := range r.labels {
			labels[l]++
		}
		for e := range r.exts {
			exts[e] = true
		}
		for m, names := range r.uses {
			for n := range names {
				g.use(imported{m, n})
			}
		}
	}
	if len(emitted) == 0 {
		return g.Response(nil), nil
	}

	var body strings.Builder
	for _, r := range emitted {
		body.WriteString("\n")
		body.WriteString(r.text)
		// GHC names a field in a DEPRECATED pragma by its label alone, so a
		// label two declarations share carries its reason in Haddock only.
		for _, p := range r.deprecated {
			if labels[p.label] == 1 {
				body.WriteString("\n" + p.pragma + "\n")
			}
		}
	}
	imports, text := g.imports(body.String(), declared)

	var b strings.Builder
	b.WriteString("-- Code generated by tdl. DO NOT EDIT.\n")
	for _, p := range pragmas {
		fmt.Fprintf(&b, "{-# LANGUAGE %s #-}\n", p)
	}
	for _, e := range slices.Sorted(maps.Keys(exts)) {
		fmt.Fprintf(&b, "{-# LANGUAGE %s #-}\n", e)
	}
	b.WriteString("\n")
	haddock(&b, "", "|", emit.Doc(&ir.Meta{Doc: g.Model.GetDoc()}))
	fmt.Fprintf(&b, "module %s where\n\n", module)
	b.WriteString(imports)
	b.WriteString(text)

	path := strings.ReplaceAll(module, ".", "/") + ".hs"
	return g.Response([]*plugin.File{{Path: path, Content: []byte(b.String())}}), nil
}

// decl renders one declaration, or nil for one that declares nothing.
func (g *generator) decl(d *ir.Decl) (*rendered, error) {
	pos := d.GetMeta().GetPosition()
	name := d.GetMeta().GetName()

	switch {
	case g.foreign[d] != imported{}:
		// Another module declares it.
		return nil, nil
	case d.GetClass() != nil:
		return g.class(d)
	case d.GetUnit() != nil:
		return nil, emit.Lost(emit.LossUnit, pos, "%s is a unit, and units are not generated yet", name)
	case d.GetPrimitive() != nil:
		// A primitive declares nothing.
		return nil, nil
	}

	typ, err := g.typeName(d)
	if err != nil {
		return nil, err
	}
	if other, ok := g.types[typ]; ok {
		return nil, emit.Unsupported(pos, "%s would declare the type %s in Haskell, and %s already does", name, typ, other)
	}
	params, err := g.params(d)
	if err != nil {
		return nil, err
	}

	r := &rendered{types: []string{typ}}
	var ctors []string
	var b strings.Builder
	haddock(&b, "", "|", emit.Doc(d.GetMeta()))
	switch e := d.GetEnumeration(); {
	case d.GetAlias() != nil:
		target, err := g.typ(d.GetAlias().GetTarget())
		if err != nil {
			return nil, err
		}
		fmt.Fprintf(&b, "type %s%s = %s\n", typ, params, target.text)
		for _, dir := range plugin.Directives(g.Target, d.GetDirectives()) {
			if dir.GetName() == "derive" {
				g.Warn(emit.Unsupported(dir.GetPosition(), "%s is an alias, which has no instances of its own to derive", name))
			}
		}
	case d.GetNewtype() != nil:
		base, err := g.typ(d.GetNewtype().GetBase())
		if err != nil {
			return nil, err
		}
		fmt.Fprintf(&b, "newtype %s%s = %s %s\n", typ, params, typ, base.arg())
		if err := g.derivings(&b, d, typ, eq, ord, show); err != nil {
			return nil, err
		}
		ctors = []string{typ}
	case d.GetStructure() != nil:
		fields, err := g.fields(r, name, d.Fields(), "  ")
		if err != nil {
			return nil, err
		}
		fmt.Fprintf(&b, "data %s%s = %s%s\n", typ, params, typ, fields)
		if err := g.derivings(&b, d, typ, eq, ord, show); err != nil {
			return nil, err
		}
		ctors = []string{typ}
	default:
		if ctors, err = g.enum(&b, r, d, typ, params, e); err != nil {
			return nil, err
		}
	}
	if reason, ok := emit.Deprecated(d.GetMeta()); ok {
		fmt.Fprintf(&b, "\n{-# DEPRECATED %s %s #-}\n", typ, str(deprecation(reason)))
	}
	if dir, ok := g.Find(d.GetDirectives(), "key"); ok {
		if err := g.keyed(&b, r, d, typ, dir, &ctors); err != nil {
			g.Warn(err)
		}
	}

	seen := map[string]bool{}
	for _, c := range ctors {
		if other, ok := g.ctors[c]; ok || seen[c] {
			if !ok {
				other = name
			}
			return nil, emit.Unsupported(pos, "%s would declare the constructor %s in Haskell, and %s already does", name, c, other)
		}
		seen[c] = true
	}
	g.types[typ] = name
	for _, c := range ctors {
		g.ctors[c] = name
	}

	g.WarnConstraints(d)
	g.warnRequires(d)
	r.text = b.String()
	return r, nil
}

// keyed renders an entity's key after it, declaring the key record's type
// and constructor when there is one.
func (g *generator) keyed(b *strings.Builder, r *rendered, d *ir.Decl, typ string, dir *ir.Directive, ctors *[]string) error {
	var kb strings.Builder
	record, labels, err := g.key(&kb, d, typ, dir)
	if err != nil {
		return err
	}
	if record != "" {
		if other, ok := g.types[record]; ok {
			return emit.Lost(emit.LossKey, dir.GetPosition(), "the key record %s would collide with %s", record, other)
		}
		r.types = append(r.types, record)
		*ctors = append(*ctors, record)
		// The record repeats the entity's labels, so a deprecated one's
		// pragma would be ambiguous.
		r.labels = append(r.labels, labels...)
	}
	b.WriteString(kb.String())
	return nil
}

// claim records the type and constructor names a declaration would take,
// before any is rendered.
func (g *generator) claim(d *ir.Decl) {
	if g.foreign[d] != (imported{}) || d.GetStructure() == nil && d.GetEnumeration() == nil && d.GetNewtype() == nil && d.GetAlias() == nil && d.GetClass() == nil {
		return
	}
	name := d.GetMeta().GetName()
	typ := g.DeclName(d, upperFirst)
	if _, ok := g.names[typ]; !ok {
		g.names[typ] = name
	}
	for _, a := range d.GetClass().GetAssocTypes() {
		if _, ok := g.names[upperFirst(a.GetMeta().GetName())]; !ok {
			g.names[upperFirst(a.GetMeta().GetName())] = name
		}
	}
	for _, v := range d.GetEnumeration().GetVariants() {
		if ctor, err := g.variantName(d, typ, v); err == nil {
			if _, ok := g.names[ctor]; !ok {
				g.names[ctor] = name
			}
		}
	}
}

// enum renders an enum as a sum type, returning its constructors. A
// fieldless enum is written on one line unless a variant carries a doc
// comment.
func (g *generator) enum(b *strings.Builder, r *rendered, d *ir.Decl, typ, params string, e *ir.Enum) ([]string, error) {
	name := d.GetMeta().GetName()
	variants := e.GetVariants()
	fielded := emit.Fielded(e)

	var ctors []string
	var lines []string
	var pragmas []string
	inline := !fielded
	// A field two variants share is one field of the enum, of one type.
	shared := map[string]string{}
	owner := map[string]string{}
	for _, v := range variants {
		ctor, err := g.variantName(d, typ, v)
		if err != nil {
			return nil, err
		}
		ctors = append(ctors, ctor)
		vname := name + "." + v.GetMeta().GetName()

		for _, f := range v.GetFields() {
			label, err := g.fieldName(vname, f)
			if err != nil {
				return nil, err
			}
			t, err := g.typ(f.GetType())
			if err != nil {
				return nil, err
			}
			if prev, ok := shared[label]; ok && prev != t.text {
				return nil, emit.Unsupported(f.GetMeta().GetPosition(),
					"%s.%s and %s are both labelled %s in Haskell with different types, and a name directive on either resolves it",
					vname, f.GetMeta().GetName(), owner[label], label)
			}
			shared[label], owner[label] = t.text, vname+"."+f.GetMeta().GetName()
		}
		fields, err := g.fields(r, vname, v.GetFields(), "      ")
		if err != nil {
			return nil, err
		}

		var vb strings.Builder
		doc := emit.Doc(v.GetMeta())
		if len(doc) > 0 {
			inline = false
			vb.WriteString("-- | " + doc[0] + "\n")
			for _, line := range doc[1:] {
				vb.WriteString("    " + strings.TrimRight("-- "+line, " ") + "\n")
			}
			vb.WriteString("    ")
		}
		vb.WriteString(ctor + fields)
		lines = append(lines, vb.String())

		if reason, ok := emit.Deprecated(v.GetMeta()); ok {
			pragmas = append(pragmas, fmt.Sprintf("{-# DEPRECATED %s %s #-}\n", ctor, str(deprecation(reason))))
		}
	}

	switch {
	case len(variants) == 0:
		fmt.Fprintf(b, "data %s%s\n", typ, params)
	case inline:
		fmt.Fprintf(b, "data %s%s = %s\n", typ, params, strings.Join(lines, " | "))
	default:
		fmt.Fprintf(b, "data %s%s\n", typ, params)
		for i, line := range lines {
			lead := "  | "
			if i == 0 {
				lead = "  = "
			}
			b.WriteString(lead + line + "\n")
		}
	}
	classes := []imported{eq, ord, show}
	if !fielded && len(variants) > 0 {
		classes = append(classes, read, enum, bounded)
	}
	if err := g.derivings(b, d, typ, classes...); err != nil {
		return nil, err
	}
	for _, p := range pragmas {
		b.WriteString("\n" + p)
	}
	return ctors, nil
}

// fields renders a record body: " {a :: A, b :: B}" on one line, or one
// field per line at indent when a field carries Haddock, or "" for none.
func (g *generator) fields(r *rendered, owner string, fields []*ir.Field, indent string) (string, error) {
	if len(fields) == 0 {
		return "", nil
	}

	type line struct {
		decl string
		doc  []string
	}
	var lines []line
	multiline := indent == "  "
	seen := map[string]bool{}
	for _, f := range fields {
		label, err := g.fieldName(owner, f)
		if err != nil {
			return "", err
		}
		if seen[label] {
			return "", emit.Unsupported(f.GetMeta().GetPosition(), "%s has two fields labelled %s in Haskell", owner, label)
		}
		seen[label] = true
		if !slices.Contains(r.labels, label) {
			r.labels = append(r.labels, label)
		}

		t, err := g.typ(f.GetType())
		if err != nil {
			return "", err
		}
		text := t.text
		if g.Tagged(f.GetDirectives(), "lazy") {
			text = "~" + t.arg()
		}

		doc := emit.Doc(f.GetMeta())
		if reason, ok := emit.Deprecated(f.GetMeta()); ok {
			if len(doc) > 0 {
				doc = append(doc, "")
			}
			if reason == "" {
				doc = append(doc, "Deprecated.")
			} else {
				doc = append(doc, "Deprecated: "+reason)
			}
			r.deprecated = append(r.deprecated, fieldPragma{
				label:  label,
				pragma: fmt.Sprintf("{-# DEPRECATED %s %s #-}", label, str(deprecation(reason))),
			})
		}
		multiline = multiline || len(doc) > 0
		lines = append(lines, line{decl: label + " :: " + text, doc: doc})
	}

	if !multiline {
		decls := make([]string, len(lines))
		for i, l := range lines {
			decls[i] = l.decl
		}
		return " {" + strings.Join(decls, ", ") + "}", nil
	}
	var b strings.Builder
	for i, l := range lines {
		lead := ", "
		if i == 0 {
			lead = "{ "
		}
		b.WriteString("\n" + indent + lead + l.decl)
		var hb strings.Builder
		haddock(&hb, indent+"  ", "^", l.doc)
		if hb.Len() > 0 {
			b.WriteString("\n" + strings.TrimSuffix(hb.String(), "\n"))
		}
	}
	b.WriteString("\n" + indent + "}")
	return b.String(), nil
}

// deriving is the stock deriving clause for the classes given.
func (g *generator) deriving(classes ...imported) string {
	names := make([]string, len(classes))
	for i, c := range classes {
		names[i] = g.use(c)
	}
	return "  deriving stock (" + strings.Join(names, ", ") + ")\n"
}

// deprecation is the message a DEPRECATED pragma carries.
func deprecation(reason string) string {
	if reason == "" {
		return "deprecated"
	}
	return reason
}

// haddock writes lines as a Haddock comment, `-- |` before what it
// documents or `-- ^` after it, or nothing when there are none.
func haddock(b *strings.Builder, indent, mark string, lines []string) {
	for i, line := range lines {
		lead := "--"
		if i == 0 {
			lead = "-- " + mark
		}
		b.WriteString(strings.TrimRight(indent+lead+" "+line, " ") + "\n")
	}
}
