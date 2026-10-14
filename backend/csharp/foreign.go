package csharp

import (
	"slices"
	"strings"

	"github.com/unstoppablemango/tdl/backend/internal/dotnet"
	"github.com/unstoppablemango/tdl/backend/internal/emit"
	"github.com/unstoppablemango/tdl/ir"
	"github.com/unstoppablemango/tdl/plugin"
)

// A `foreign` directive maps a declaration, a primitive included, to a type
// another assembly or namespace declares, which the generated code names
// instead of declaring. An extern without one is the type its own package's
// csharp target block generates.

// planForeign reads every mapping before rendering: each declaration's,
// the prelude's included, and each extern's, from a mapping or else from
// its dependency's csharp target block.
func (g *generator) planForeign() {
	g.foreign = map[*ir.Decl]string{}
	g.externs = map[*ir.Extern]string{}

	for _, d := range g.Model.GetDecls() {
		t, ok := g.mapping(d.GetDirectives(), d.GetMeta().GetName())
		if !ok {
			continue
		}
		g.foreign[d] = t
		if !emit.IsOwn(d) {
			continue
		}
		g.warnForeignConstraints(d)
		if key, ok := g.Find(d.GetDirectives(), "key"); ok {
			g.Lossy(emit.LossKey, key.GetPosition(),
				"%s is a foreign type, so its key is not generated: the key method needs the type's properties", d.GetMeta().GetName())
		}
	}

	for _, e := range g.Model.GetExterns() {
		if t, ok := g.mapping(e.GetDirectives(), e.GetPackage()+"."+e.GetName()); ok {
			g.externs[e] = t
		} else if t, ok := g.dependency(e); ok {
			g.externs[e] = t
		}
	}
}

// mapping reads the foreign directive among dirs as the type it names,
// warning when it is unusable.
func (g *generator) mapping(dirs []*ir.Directive, of string) (string, bool) {
	dir, ok := g.Find(dirs, "foreign")
	if !ok {
		return "", false
	}
	q, err := dotnet.Parse(texts(dir))
	if err != nil {
		g.Warn(emit.Unsupported(dir.GetPosition(), "the foreign mapping of %s %v", of, err))
		return "", false
	}
	return qualify(q.Namespace, q.Name), true
}

// dependency is the type an extern's own package generates, when that
// package has a csharp target block: in the block's namespace, or else the
// one its package derives, under the extern's `name` or its own name with
// the first letter upper-cased.
func (g *generator) dependency(e *ir.Extern) (string, bool) {
	for _, dep := range g.Model.GetImports() {
		if dep.GetPackage() != e.GetPackage() {
			continue
		}
		if len(plugin.Directives(g.Target, dep.GetDirectives())) == 0 {
			return "", false
		}
		ns, ok := g.Text(dep.GetDirectives(), "namespace")
		if !ok {
			ns = dotnet.Namespace(dep.GetPackage())
		}
		name, ok := g.Text(e.GetDirectives(), "name")
		if !ok {
			name = upperFirst(e.GetName())
		}
		q, err := dotnet.Parse([]string{ns, name})
		if err != nil {
			g.Warn(emit.Unsupported(e.GetPosition(), "%s.%s is generated as a type that %v", e.GetPackage(), e.GetName(), err))
			return "", false
		}
		return qualify(q.Namespace, q.Name), true
	}
	return "", false
}

// warnForeignConstraints warns at each constraint on a foreign declaration,
// since the backend does not know what the type holds.
func (g *generator) warnForeignConstraints(d *ir.Decl) {
	cs := slices.Clone(d.GetNewtype().GetValueConstraints())
	fields := slices.Clone(d.Fields())
	for _, v := range d.GetEnumeration().GetVariants() {
		fields = append(fields, v.GetFields()...)
	}
	for _, f := range fields {
		cs = append(cs, f.GetConstraints()...)
	}
	for _, c := range cs {
		g.Lossy(emit.LossConstraint, c.GetPosition(),
			"%s is not checked: %s is a foreign type, whose values another assembly decides", emit.ConstraintText(c), d.GetMeta().GetName())
	}
}

// foreignPrim is the type a primitive is mapped to, if any.
func (g *generator) foreignPrim(id *ir.ID) (string, bool) {
	d := g.Model.Decl(g.Model.Type(id).GetCtor())
	t, ok := g.foreign[d]
	return t, ok
}

// attributesOf renders every attribute directive among dirs, warning at one
// that is unusable.
func (g *generator) attributesOf(dirs []*ir.Directive) []string {
	var out []string
	for _, dir := range plugin.Directives(g.Target, dirs) {
		if dir.GetName() != "attribute" {
			continue
		}
		q, err := dotnet.Parse(texts(dir))
		if err != nil {
			g.Warn(emit.Unsupported(dir.GetPosition(), "attribute %v", err))
			continue
		}
		out = append(out, "["+qualify(q.Namespace, q.Name)+"]")
	}
	return out
}

// blockDirectives are the directives written on the target block itself.
func (g *generator) blockDirectives() []*ir.Directive {
	var out []*ir.Directive
	for _, block := range g.Model.GetTargets() {
		if block.GetMeta().GetName() == g.Target {
			out = append(out, block.GetDirectives()...)
		}
	}
	return out
}

// qualify writes a type another namespace declares from the global
// namespace, escaping each segment C# reserves.
func qualify(ns, name string) string {
	segs := strings.Split(ns, ".")
	for i, s := range segs {
		segs[i] = escape(s)
	}
	return "global::" + strings.Join(segs, ".") + "." + escape(name)
}

// texts is a directive's arguments as written.
func texts(dir *ir.Directive) []string {
	out := make([]string, len(dir.GetArgs()))
	for i, a := range dir.GetArgs() {
		out[i] = a.GetText()
	}
	return out
}
