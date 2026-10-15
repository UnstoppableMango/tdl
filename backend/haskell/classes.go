package haskell

import (
	"cmp"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/unstoppablemango/tdl/backend/internal/emit"
	"github.com/unstoppablemango/tdl/ir"
)

// A TDL class is a type class. A field it requires is a HasField
// superclass rather than a method, since a method is a module-wide name and
// two classes requiring one field would collide. Each declared conformance
// and each instance is an instance declaration, and so is each instance a
// required class implies, since Haskell asks for every superclass instance
// where TDL closes over them.

var hasField = imported{"GHC.Records", "HasField"}

// classVars is a class's type variables: its parameters', or `a` for a
// class declared without parameters, whose one parameter is the type
// satisfying it.
func classVars(d *ir.Decl) []string {
	ps := d.Params()
	if len(ps) == 0 {
		return []string{"a"}
	}
	vars := make([]string, len(ps))
	for i, p := range ps {
		vars[i] = tyvar(p.GetName())
	}
	return vars
}

// arity is how many arguments a class takes in Haskell.
func arity(d *ir.Decl) int { return max(1, len(d.Params())) }

// class renders a class as a type class whose context holds the classes it
// requires, the constraints on its parameters, and one HasField per field.
func (g *generator) class(d *ir.Decl) (*rendered, error) {
	pos := d.GetMeta().GetPosition()
	name := d.GetMeta().GetName()
	c := d.GetClass()

	typ, err := g.typeName(d)
	if err != nil {
		return nil, err
	}
	if other, ok := g.types[typ]; ok {
		return nil, emit.Unsupported(pos, "%s would declare the class %s in Haskell, and %s already does", name, typ, other)
	}
	params := " a"
	if len(d.Params()) > 0 {
		if params, err = g.params(d); err != nil {
			return nil, err
		}
	}
	vars := classVars(d)
	if len(c.GetFields()) > 0 && len(vars) > 1 {
		return nil, emit.Lost(emit.LossClass, pos, "%s takes several parameters and requires fields, and no one of its parameters is the type holding them", name)
	}
	self := head{args: vars}

	var ctx []string
	refs := append(slices.Clone(c.GetRequiresClasses()), c.GetConstraints()...)
	for _, ref := range refs {
		text, err := g.constraint(ref, self, nil)
		if err != nil {
			g.Warn(dropped(err, name))
			continue
		}
		ctx = append(ctx, text)
	}
	seen := map[string]bool{}
	for _, f := range c.GetFields() {
		label, err := g.fieldName(name, f)
		if err != nil {
			return nil, err
		}
		if seen[label] {
			return nil, emit.Unsupported(f.GetMeta().GetPosition(), "%s has two fields labelled %s in Haskell", name, label)
		}
		seen[label] = true
		t, err := g.typ(f.GetType())
		if err != nil {
			return nil, err
		}
		g.needsTypes([]*ir.ID{f.GetType()})
		g.exts["DataKinds"] = true
		ctx = append(ctx, fmt.Sprintf("%s %s %s %s", g.use(hasField), str(label), vars[0], t.arg()))
	}

	var deps []string
	for _, fd := range c.GetFunDeps() {
		from, err := depVars(d, fd.GetFrom(), fd.GetPosition())
		if err != nil {
			return nil, err
		}
		to, err := depVars(d, fd.GetTo(), fd.GetPosition())
		if err != nil {
			return nil, err
		}
		deps = append(deps, from+" -> "+to)
	}

	var assoc []string
	types := []string{typ}
	for _, at := range c.GetAssocTypes() {
		an := upperFirst(at.GetMeta().GetName())
		apos := at.GetMeta().GetPosition()
		switch other, ok := g.types[an]; {
		case !conid.MatchString(an):
			return nil, emit.Unsupported(apos, "%s.%s would be named %s in Haskell, which cannot name a type", name, at.GetMeta().GetName(), an)
		case ok || slices.Contains(types, an):
			if !ok {
				other = name
			}
			return nil, emit.Unsupported(apos, "%s.%s would declare the type family %s in Haskell, and %s already does", name, at.GetMeta().GetName(), an, other)
		case unitKinded(at.GetKind()):
			return nil, emit.Lost(emit.LossUnit, apos, "%s.%s is a unit, and units are not generated yet", name, at.GetMeta().GetName())
		}
		types = append(types, an)
		var ab strings.Builder
		haddock(&ab, "  ", "|", emit.Doc(at.GetMeta()))
		ab.WriteString("  type " + an + " " + strings.Join(vars, " "))
		if k := at.GetKind(); k != nil && (k.GetArrow() != nil || k.GetParen() != nil) {
			ab.WriteString(" :: " + g.kind(k))
		}
		assoc = append(assoc, ab.String()+"\n")
		g.exts["TypeFamilies"] = true
	}

	var b strings.Builder
	haddock(&b, "", "|", emit.Doc(d.GetMeta()))
	b.WriteString("class ")
	if len(ctx) > 0 {
		b.WriteString("(" + strings.Join(ctx, ", ") + ") => ")
	}
	b.WriteString(typ + params)
	if len(deps) > 0 {
		g.exts["FunctionalDependencies"] = true
		b.WriteString(" | " + strings.Join(deps, ", "))
	}
	if len(assoc) > 0 {
		b.WriteString(" where\n")
		for _, a := range assoc {
			b.WriteString(a)
		}
	} else {
		b.WriteString("\n")
	}
	if reason, ok := emit.Deprecated(d.GetMeta()); ok {
		fmt.Fprintf(&b, "\n{-# DEPRECATED %s %s #-}\n", typ, str(deprecation(reason)))
	}
	if g.superCycle(d) {
		g.exts["UndecidableSuperClasses"] = true
	}

	for _, t := range types {
		g.types[t] = name
	}
	return &rendered{text: b.String(), types: types}, nil
}

// depVars is one side of a functional dependency as type variables.
func depVars(d *ir.Decl, names []string, pos *ir.Position) (string, error) {
	vars := make([]string, len(names))
	for i, n := range names {
		if !slices.ContainsFunc(d.Params(), func(p *ir.Param) bool { return p.GetName() == n }) {
			return "", emit.Unsupported(pos, "the functional dependency of %s names %s, which is not one of its parameters", d.GetMeta().GetName(), n)
		}
		vars[i] = tyvar(n)
	}
	return strings.Join(vars, " "), nil
}

// dropped is the warning for a constraint a context is written without.
func dropped(err error, of string) error {
	var u *emit.UnsupportedError
	if !errors.As(err, &u) {
		return emit.Lost(emit.LossClass, nil, "%s is generated without a constraint: %v", of, err)
	}
	return emit.Lost(emit.LossClass, u.Position, "%s is generated without a constraint: %s", of, u.What)
}

// superCycle reports whether a class reaches itself through the classes it
// requires, which GHC accepts only under UndecidableSuperClasses.
func (g *generator) superCycle(start *ir.Decl) bool {
	seen := map[*ir.Decl]bool{}
	var reaches func(*ir.Decl) bool
	reaches = func(d *ir.Decl) bool {
		for _, ref := range append(slices.Clone(d.GetClass().GetRequiresClasses()), d.GetClass().GetConstraints()...) {
			next := g.Model.Decl(ref.GetClass())
			if next == nil || next.GetClass() == nil || !emit.IsOwn(next) {
				continue
			}
			if next == start {
				return true
			}
			if !seen[next] {
				seen[next] = true
				if reaches(next) {
					return true
				}
			}
		}
		return false
	}
	return reaches(start)
}

// head is the arguments an instance applies its class to, each as an
// argument. ids holds them as type references when the instance wrote
// them, and is nil for a declared conformance, whose head is the
// declaration applied to its own type variables.
type head struct {
	args []string
	ids  []bound
}

func (h head) String() string { return strings.Join(h.args, " ") }

// classOf is the class a reference names, when this module generates it.
// An instance or constraint naming a class in another package or the
// prelude's Entity is not generated: the other package's class may require
// instances this module cannot see, and Entity has no Haskell class.
func (g *generator) classOf(ref *ir.ClassRef) (*ir.Decl, string, error) {
	pos := ref.GetPosition()
	if e := ref.GetExtern(); e != nil {
		name := e.GetName()
		if i := int(e.GetIndex()); i >= 0 && i < len(g.Model.GetExterns()) {
			x := g.Model.GetExterns()[i]
			name = x.GetPackage() + "." + x.GetName()
		}
		return nil, "", emit.Lost(emit.LossClass, pos, "%s is a class in another package, whose own requirements this module cannot see", name)
	}
	d := g.Model.Decl(ref.GetClass())
	switch {
	case d == nil || d.GetClass() == nil:
		return nil, "", emit.Unsupported(pos, "class %s did not resolve", ref.GetClass().GetName())
	case !emit.IsOwn(d):
		return nil, "", emit.Lost(emit.LossClass, pos, "%s is the prelude's, and has no Haskell class", d.GetMeta().GetName())
	}
	g.needs = append(g.needs, d)
	if f, ok := g.foreign[d]; ok {
		return d, g.use(f), nil
	}
	return d, g.DeclName(d, upperFirst), nil
}

// constraint renders a class reference applied to its arguments. A
// reference written without the class's first argument, as a conformance,
// a required class, or `requires Entity` on a class writes it, applies it
// to h's first argument, which is the type satisfying the class. A type
// parameter in its arguments is the parameter of the class in, which h
// supplies; with in nil, it is the declaration's own type variable.
func (g *generator) constraint(ref *ir.ClassRef, h head, in *ir.Decl) (string, error) {
	class, name, err := g.classOf(ref)
	if err != nil {
		return "", err
	}
	args, err := g.classArgs(ref, h, in)
	if err != nil {
		return "", err
	}
	if len(args) < arity(class) && len(h.args) > 0 {
		args = append([]string{h.args[0]}, args...)
	}
	if len(args) != arity(class) {
		return "", emit.Unsupported(ref.GetPosition(), "%s takes %d argument(s) in Haskell, and %d were given", class.GetMeta().GetName(), arity(class), len(args))
	}
	return name + " " + strings.Join(args, " "), nil
}

// classArgs renders a reference's arguments, substituting h's for the
// parameters of the class in.
func (g *generator) classArgs(ref *ir.ClassRef, h head, in *ir.Decl) ([]string, error) {
	out := make([]string, len(ref.GetArgs()))
	for i, id := range ref.GetArgs() {
		if in == nil {
			t, err := g.typ(id)
			if err != nil {
				return nil, err
			}
			out[i] = t.arg()
			continue
		}
		if h.ids != nil {
			t, err := g.render(bound{id, &frame{h.ids}})
			if err != nil {
				return nil, err
			}
			out[i] = t.arg()
			continue
		}
		t := g.Model.Type(id)
		p := t.GetParam()
		switch {
		case p != nil && len(t.GetArgs()) == 0 && int(p.GetIndex()) < len(h.args):
			out[i] = h.args[p.GetIndex()]
		case p != nil || g.mentionsParam(id):
			return nil, emit.Unsupported(ref.GetPosition(), "%s applies a parameter of %s, which a conformance cannot substitute", ref.GetClass().GetName(), in.GetMeta().GetName())
		default:
			r, err := g.typ(id)
			if err != nil {
				return nil, err
			}
			out[i] = r.arg()
		}
	}
	return out, nil
}

// mentionsParam reports whether a type reference holds a type parameter.
func (g *generator) mentionsParam(id *ir.ID) bool {
	t := g.Model.Type(id)
	if t.GetParam() != nil {
		return true
	}
	return slices.ContainsFunc(t.GetArgs(), g.mentionsParam)
}

// instance is one instance declaration the module may write.
type instance struct {
	class *ir.Decl
	h     head
	key   string
	// reqs is the context it was written with, and subject the declaration
	// its head applies, if any.
	reqs    []*ir.ClassRef
	subject *ir.Decl
	// after is the declaration whose conformance it is, written after it,
	// or nil for an instance written after every declaration.
	after *ir.Decl
	r     *rendered
	// requires is the key of each instance it needs: one per class its
	// class requires, and one per constraint on its class's parameters.
	requires []string
	pos      *ir.Position
	what     string
	implied  bool
}

// instances plans every instance the module writes: each declared
// conformance, each separately declared instance, and each instance a
// required class implies for either. An instance whose class or head is
// not generated, or whose class requires an instance the module does not
// write, warns and is dropped.
func (g *generator) instances(own []*ir.Decl, skipped map[*ir.Decl]bool) []*instance {
	var out []*instance
	keys := map[string]bool{}
	add := func(inst *instance) bool {
		if keys[inst.key] {
			return false
		}
		keys[inst.key] = true
		out = append(out, inst)
		return true
	}

	// A separate instance carries associated type bindings, so it wins over
	// a conformance with the same head.
	type pending struct {
		i     *instance
		binds []*ir.AssocBind
	}
	var explicit []pending
	for _, inst := range g.Model.GetInstances() {
		i, err := g.separate(inst)
		switch {
		case err != nil:
			g.Warn(err)
		case i != nil && add(i):
			explicit = append(explicit, pending{i, inst.GetBinds()})
		}
	}
	for _, d := range own {
		if skipped[d] || g.foreign[d] != (imported{}) || d.GetStructure() == nil && d.GetEnumeration() == nil {
			continue
		}
		for _, ref := range conforms(d) {
			i, err := g.conformance(d, ref)
			switch {
			case err != nil:
				g.Warn(err)
			case i != nil && add(i):
				explicit = append(explicit, pending{i, nil})
			}
		}
	}
	for _, p := range explicit {
		g.implied(p.i, p.binds, add)
	}

	// Drop each instance naming what is not generated, then each requiring
	// a dropped or missing one, until nothing changes.
	kept := map[*instance]bool{}
	for _, inst := range out {
		kept[inst] = true
		for _, n := range inst.r.needs {
			if skipped[n] {
				g.Warn(emit.Unsupported(inst.pos, "%s names %s, which is not generated", inst.what, n.GetMeta().GetName()))
				kept[inst] = false
				break
			}
		}
	}
	for changed := true; changed; {
		changed = false
		have := map[string]bool{}
		for _, inst := range out {
			if kept[inst] {
				have[inst.key] = true
			}
		}
		for _, inst := range out {
			if !kept[inst] {
				continue
			}
			for _, req := range inst.requires {
				if !have[req] {
					g.Warn(emit.Lost(emit.LossClass, inst.pos, "%s needs the instance %s, which is not generated", inst.what, plain(req)))
					kept[inst] = false
					changed = true
					break
				}
			}
		}
	}
	return slices.DeleteFunc(out, func(i *instance) bool { return !kept[i] })
}

// plain is rendered text with its placeholders written unqualified, for a
// message.
func plain(s string) string { return placeholder.ReplaceAllString(s, "$2") }

// conforms is the classes a declaration says it satisfies.
func conforms(d *ir.Decl) []*ir.ClassRef {
	if s := d.GetStructure(); s != nil {
		return s.GetConforms()
	}
	return d.GetEnumeration().GetConforms()
}

// entity reports whether a reference names the prelude's Entity, whose
// conformance is identity rather than an instance.
func (g *generator) entity(ref *ir.ClassRef) bool {
	c := g.Model.Decl(ref.GetClass())
	return ref.GetExtern() == nil && c != nil && !emit.IsOwn(c)
}

// conformance plans the instance a declaration's conformance declares, or
// nil for a conformance to the prelude's Entity.
func (g *generator) conformance(d *ir.Decl, ref *ir.ClassRef) (*instance, error) {
	if g.entity(ref) {
		return nil, nil
	}
	typ, err := g.typeName(d)
	if err != nil {
		return nil, err
	}
	at := ref.GetPosition()
	if at == nil {
		at = d.GetMeta().GetPosition()
	}
	g.begin()
	class, _, err := g.classOf(ref)
	if err != nil {
		return nil, err
	}
	args, err := g.classArgs(ref, head{}, nil)
	if err != nil {
		return nil, err
	}
	g.needsTypes(ref.GetArgs())
	h := head{args: append([]string{applyTo(typ, d).arg()}, args...)}
	what := fmt.Sprintf("%s's conformance to %s", d.GetMeta().GetName(), class.GetMeta().GetName())
	inst := &instance{class: class, h: h, subject: d, after: d, pos: at, what: what}
	if err := g.instance(inst, nil, nil); err != nil {
		return nil, err
	}
	return inst, nil
}

// separate plans a separately declared instance, or nil for an instance of
// the prelude's Entity.
func (g *generator) separate(decl *ir.Instance) (*instance, error) {
	ref := decl.GetClass()
	pos := decl.GetMeta().GetPosition()
	if g.entity(ref) {
		return nil, nil
	}
	g.begin()
	class, _, err := g.classOf(ref)
	if err != nil {
		return nil, err
	}
	what := "the instance of " + class.GetMeta().GetName()
	seen := map[string]string{}
	for _, p := range decl.GetParams() {
		v := tyvar(p.GetName())
		switch other, dup := seen[v]; {
		case unitKinded(p.GetKind()):
			return nil, emit.Lost(emit.LossUnit, pos, "type parameter %s of %s is a unit, and units are not generated yet", p.GetName(), what)
		case keywords[v] || !varid.MatchString(v):
			return nil, emit.Lost(emit.LossGeneric, pos, "type parameter %s of %s would be the type variable %s, which Haskell refuses", p.GetName(), what, v)
		case dup:
			return nil, emit.Lost(emit.LossGeneric, pos, "type parameters %s and %s of %s are both the type variable %s", other, p.GetName(), what, v)
		}
		seen[v] = p.GetName()
	}
	h := head{ids: bounds(ref.GetArgs(), nil)}
	for _, id := range ref.GetArgs() {
		t, err := g.typ(id)
		if err != nil {
			return nil, err
		}
		h.args = append(h.args, t.arg())
	}
	g.needsTypes(ref.GetArgs())
	if len(h.args) != arity(class) {
		return nil, emit.Unsupported(pos, "%s applies it to %d argument(s), and it takes %d in Haskell", what, len(h.args), arity(class))
	}
	inst := &instance{class: class, h: h, reqs: decl.GetRequires(), pos: pos, what: "the instance " + class.GetMeta().GetName() + " " + plain(h.String())}
	if t := g.Model.Type(h.ids[0].id); t != nil {
		inst.subject = g.Model.Decl(t.GetCtor())
	}
	if err := g.instance(inst, decl.GetBinds(), emit.Doc(decl.GetMeta())); err != nil {
		return nil, err
	}
	return inst, nil
}

// needsTypes records the declarations type references name.
func (g *generator) needsTypes(ids []*ir.ID) {
	g.needs = append(g.needs, g.TypeReferences(ids...)...)
}

// begin resets what the next render records.
func (g *generator) begin() {
	g.uses = map[string]map[string]bool{}
	g.exts = map[string]bool{}
	g.needs = nil
}

// quietly runs f without recording what it renders.
func (g *generator) quietly(f func()) {
	uses, exts, needs := g.uses, g.exts, g.needs
	g.uses, g.exts = map[string]map[string]bool{}, map[string]bool{}
	f()
	g.uses, g.exts, g.needs = uses, exts, needs
}

// instance renders inst under its context, with the bindings for the
// associated types its class declares, after begin.
func (g *generator) instance(inst *instance, binds []*ir.AssocBind, doc []string) error {
	class, h, pos, what := inst.class, inst.h, inst.pos, inst.what
	name := g.DeclName(class, upperFirst)
	if f, ok := g.foreign[class]; ok {
		name = g.use(f)
	}
	// Instances are told apart with synonyms expanded, as GHC does, so
	// one for an alias and one for its target are one instance.
	inst.key = name + " " + h.String()
	if h.ids != nil {
		g.quietly(func() {
			g.expand = true
			defer func() { g.expand = false }()
			args := make([]string, len(h.ids))
			for i, b := range h.ids {
				t, err := g.render(b)
				if err != nil {
					return
				}
				args[i] = t.arg()
			}
			inst.key = name + " " + strings.Join(args, " ")
		})
	}
	// The head is rendered once, and an implied instance writes it again.
	for _, ref := range placeholder.FindAllStringSubmatch(h.String(), -1) {
		g.use(imported{ref[1], ref[2]})
	}
	if h.ids != nil && g.hasFields(class) {
		if a, err := g.resolve(h.ids[0]); err == nil && a.t.GetExtern() != nil {
			// HasField finds a field only when it is in scope.
			if f, ok := g.extern(a.t); ok && f.module != g.self {
				g.uses[f.module][f.name+" (..)"] = true
			}
		}
	}

	if s := inst.subject; s != nil && emit.IsOwn(s) && g.hasFields(class) {
		if f, ok := g.foreign[s]; ok {
			return emit.Lost(emit.LossClass, pos, "%s is for %s, which is the foreign type %s.%s, whose fields this module cannot see", what, s.GetMeta().GetName(), f.module, f.name)
		}
		if err := g.labelsAgree(class, s, pos, what, map[*ir.Decl]bool{}); err != nil {
			return err
		}
	}

	var ctx []string
	for _, req := range inst.reqs {
		text, err := g.constraint(req, head{}, nil)
		if err != nil {
			if !inst.implied {
				g.Warn(dropped(err, what))
			}
			continue
		}
		ctx = append(ctx, text)
		g.needsTypes(req.GetArgs())
	}

	// Each class its class requires and each constraint on its class's
	// parameters needs an instance at the same head.
	// A class the class itself is written without needs none, and an
	// instance the backend cannot name drops this one.
	var missing error
	g.quietly(func() {
		g.expand = true
		defer func() { g.expand = false }()
		refs := append(slices.Clone(class.GetClass().GetRequiresClasses()), class.GetClass().GetConstraints()...)
		for _, ref := range refs {
			if _, _, err := g.classOf(ref); err != nil {
				continue
			}
			text, err := g.constraint(ref, h, class)
			if err != nil {
				missing = cmp.Or(missing, err)
				continue
			}
			inst.requires = append(inst.requires, text)
		}
	})
	if missing != nil {
		return emit.Lost(emit.LossClass, pos, "%s needs an instance this module cannot name: %v", what, missing)
	}

	var body []string
	bound := map[string]bool{}
	for _, b := range binds {
		if !slices.ContainsFunc(class.GetClass().GetAssocTypes(), func(a *ir.AssocType) bool { return a.GetMeta().GetName() == b.GetName() }) {
			continue // a required class's, written on its instance
		}
		bound[b.GetName()] = true
		t, err := g.typ(b.GetType())
		if err != nil {
			return err
		}
		g.needsTypes([]*ir.ID{b.GetType()})
		body = append(body, fmt.Sprintf("  type %s %s = %s\n", upperFirst(b.GetName()), h, t.text))
	}
	for _, a := range class.GetClass().GetAssocTypes() {
		if !bound[a.GetMeta().GetName()] {
			return emit.Lost(emit.LossClass, pos, "%s binds no type for %s.%s", what, class.GetMeta().GetName(), a.GetMeta().GetName())
		}
	}

	var b strings.Builder
	haddock(&b, "", "|", doc)
	b.WriteString("instance ")
	if len(ctx) > 0 {
		b.WriteString("(" + strings.Join(ctx, ", ") + ") => ")
		if undecidable(h, ctx) {
			g.exts["UndecidableInstances"] = true
		}
	}
	b.WriteString(name + " " + h.String())
	if len(body) > 0 {
		g.exts["TypeFamilies"] = true
		b.WriteString(" where\n" + strings.Join(body, ""))
	} else {
		b.WriteString("\n")
	}
	inst.r = &rendered{text: b.String(), uses: g.uses, exts: g.exts, needs: g.needs}
	return nil
}

// implied plans the instance each class i's class requires implies at the
// same head and under the same context, transitively, binding the
// associated types each declares from binds.
func (g *generator) implied(i *instance, binds []*ir.AssocBind, add func(*instance) bool) {
	for _, ref := range i.class.GetClass().GetRequiresClasses() {
		g.begin()
		super, _, err := g.classOf(ref)
		if err != nil {
			continue // the class is written without it
		}
		args, err := g.classArgs(ref, i.h, i.class)
		if err != nil {
			continue
		}
		h := head{args: args}
		if i.h.ids != nil {
			h.ids = bounds(ref.GetArgs(), &frame{i.h.ids})
		}
		if len(args) < arity(super) {
			h.args = append([]string{i.h.args[0]}, h.args...)
			if h.ids != nil {
				h.ids = append([]bound{i.h.ids[0]}, h.ids...)
			}
		}
		// The head's own references and the instance's.
		g.needs = append(g.needs, i.r.needs...)
		inst := &instance{
			class: super, h: h, reqs: i.reqs, subject: i.subject, after: i.after, pos: i.pos, implied: true,
			what: fmt.Sprintf("the instance %s %s, which %s implies", super.GetMeta().GetName(), plain(h.String()), i.what),
		}
		if err := g.instance(inst, binds, nil); err != nil {
			g.Warn(err)
			continue
		}
		if add(inst) {
			g.implied(inst, binds, add)
		}
	}
}

// hasFields reports whether a class, or a class it requires, requires a
// field.
func (g *generator) hasFields(class *ir.Decl) bool {
	seen := map[*ir.Decl]bool{}
	var walk func(*ir.Decl) bool
	walk = func(d *ir.Decl) bool {
		if d == nil || d.GetClass() == nil || seen[d] {
			return false
		}
		seen[d] = true
		if len(d.GetClass().GetFields()) > 0 {
			return true
		}
		for _, ref := range d.GetClass().GetRequiresClasses() {
			if walk(g.Model.Decl(ref.GetClass())) {
				return true
			}
		}
		return false
	}
	return walk(class)
}

// labelsAgree checks that each field a class requires, itself or through a
// class it requires, has the label on the declaration satisfying it that
// the class's HasField names, which a name directive on either can break.
func (g *generator) labelsAgree(class, subject *ir.Decl, pos *ir.Position, what string, seen map[*ir.Decl]bool) error {
	if class == nil || class.GetClass() == nil || seen[class] {
		return nil
	}
	seen[class] = true
	for _, f := range class.GetClass().GetFields() {
		i := slices.IndexFunc(subject.Fields(), func(s *ir.Field) bool { return s.GetMeta().GetName() == f.GetMeta().GetName() })
		if i < 0 {
			continue
		}
		want, err := g.fieldName(class.GetMeta().GetName(), f)
		if err != nil {
			return err
		}
		got, err := g.fieldName(subject.GetMeta().GetName(), subject.Fields()[i])
		if err != nil {
			return err
		}
		if got != want {
			return emit.Lost(emit.LossClass, pos, "%s needs %s.%s labelled %s, as %s labels it, and it is labelled %s",
				what, subject.GetMeta().GetName(), f.GetMeta().GetName(), want, class.GetMeta().GetName(), got)
		}
	}
	for _, ref := range class.GetClass().GetRequiresClasses() {
		if err := g.labelsAgree(g.Model.Decl(ref.GetClass()), subject, pos, what, seen); err != nil {
			return err
		}
	}
	return nil
}

// undecidable reports whether an instance's context breaks the Paterson
// conditions: a constraint no smaller than the head, or one mentioning a
// type variable more often than the head does.
func undecidable(h head, ctx []string) bool {
	size, vars := measureText(h.String())
	for _, c := range ctx {
		_, rest, _ := strings.Cut(plain(c), " ")
		n, vs := measureText(rest)
		if n >= size {
			return true
		}
		for v, k := range vs {
			if k > vars[v] {
				return true
			}
		}
	}
	return false
}

var word = regexp.MustCompile(`[A-Za-z_][A-Za-z0-9_'.]*|\[`)

// measureText counts the constructors and variables in rendered types, and
// each variable's occurrences.
func measureText(s string) (int, map[string]int) {
	vars := map[string]int{}
	words := word.FindAllString(plain(s), -1)
	for _, w := range words {
		if w[0] >= 'a' && w[0] <= 'z' || w[0] == '_' {
			vars[w]++
		}
	}
	return len(words), vars
}

// warnRequires warns at each constraint on a struct's, an enum's, or a
// newtype's parameters. Haskell deprecated datatype contexts, so the
// constraint is written on the functions generated for the type instead,
// and a consumer can build a value the model forbids.
func (g *generator) warnRequires(d *ir.Decl) {
	name := d.GetMeta().GetName()
	for _, ref := range d.Constraints() {
		at := ref.GetPosition()
		if at == nil {
			at = d.GetMeta().GetPosition()
		}
		var text string
		var err error
		g.quietly(func() { text, err = g.constraint(ref, head{}, nil) })
		if err != nil {
			g.Warn(dropped(err, name))
			continue
		}
		g.Lossy(emit.LossClass, at, "%s requires %s of its parameters, which a Haskell type cannot carry, so only the functions generated for it require it", name, plain(text))
	}
}

// requiresContext is the context the functions generated for a declaration
// carry: its `requires` clause, written as "(C t) => ", or "" for none.
func (g *generator) requiresContext(d *ir.Decl) string {
	var ctx []string
	for _, ref := range d.Constraints() {
		if text, err := g.constraint(ref, head{}, nil); err == nil {
			ctx = append(ctx, text)
		}
	}
	if len(ctx) == 0 {
		return ""
	}
	return "(" + strings.Join(ctx, ", ") + ") => "
}
