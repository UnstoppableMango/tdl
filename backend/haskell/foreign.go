package haskell

import (
	"slices"
	"strings"

	"github.com/unstoppablemango/tdl/backend/internal/emit"
	"github.com/unstoppablemango/tdl/ir"
	"github.com/unstoppablemango/tdl/plugin"
)

// A `foreign` directive maps a declaration to a type another module
// declares, which the generated module imports instead of declaring. It is
// how a model replaces the uuid and decimal placeholders.

// planForeign reads every mapping before rendering: each declaration's,
// the prelude's included, and each extern's, from a mapping or else from
// its dependency's haskell target block.
func (g *generator) planForeign() {
	g.foreign = map[*ir.Decl]imported{}
	g.externs = map[*ir.Extern]imported{}

	for _, d := range g.Model.GetDecls() {
		f, ok := g.mapping(d.GetDirectives(), d.GetMeta().GetName())
		if !ok {
			continue
		}
		g.foreign[d] = f
		if !emit.IsOwn(d) {
			continue
		}
		g.warnForeignConstraints(d)
		if key, ok := g.Find(d.GetDirectives(), "key"); ok {
			g.Lossy(emit.LossKey, key.GetPosition(),
				"%s is a foreign type, so its key is not generated: the key function needs the type's fields", d.GetMeta().GetName())
		}
	}

	for _, e := range g.Model.GetExterns() {
		if f, ok := g.mapping(e.GetDirectives(), e.GetPackage()+"."+e.GetName()); ok {
			g.externs[e] = f
		} else if f, ok := g.dependency(e); ok {
			g.externs[e] = f
		}
	}
}

// mapping reads the foreign directive among dirs, warning when it is
// unusable.
func (g *generator) mapping(dirs []*ir.Directive, of string) (imported, bool) {
	dir, ok := g.Find(dirs, "foreign")
	if !ok {
		return imported{}, false
	}
	args := dir.GetArgs()
	if len(args) != 2 {
		g.Warn(emit.Unsupported(dir.GetPosition(),
			"foreign takes a module and a type name, and %d argument(s) were given", len(args)))
		return imported{}, false
	}
	f := imported{module: args[0].GetText(), name: args[1].GetText()}
	switch {
	case !moduleName(f.module):
		g.Warn(emit.Unsupported(dir.GetPosition(), "%s is mapped into %q, which is not a Haskell module name", of, f.module))
		return imported{}, false
	case !conid.MatchString(f.name):
		g.Warn(emit.Unsupported(dir.GetPosition(), "%s is mapped to %q, which is not a Haskell type name", of, f.name))
		return imported{}, false
	}
	return f, true
}

// dependency is the type an extern's own package generates, when that
// package has a haskell target block: in the block's module, or else the
// module its package names, under the extern's `name` or its own name with
// the first letter upper-cased.
func (g *generator) dependency(e *ir.Extern) (imported, bool) {
	for _, dep := range g.Model.GetImports() {
		if dep.GetPackage() != e.GetPackage() {
			continue
		}
		if len(plugin.Directives(g.Target, dep.GetDirectives())) == 0 {
			return imported{}, false
		}
		module, ok := g.Text(dep.GetDirectives(), "module")
		if !ok {
			module = packageModule(dep.GetPackage())
		}
		name, ok := g.Text(e.GetDirectives(), "name")
		if !ok {
			name = upperFirst(e.GetName())
		}
		of := e.GetPackage() + "." + e.GetName()
		switch {
		case !moduleName(module):
			g.Warn(emit.Unsupported(e.GetPosition(), "%s is generated into %q, which is not a Haskell module name", of, module))
			return imported{}, false
		case !conid.MatchString(name):
			g.Warn(emit.Unsupported(e.GetPosition(), "%s is generated as %q, which is not a Haskell type name", of, name))
			return imported{}, false
		}
		return imported{module: module, name: name}, true
	}
	return imported{}, false
}

// extern is the mapping of the extern a type refers to, if any.
func (g *generator) extern(t *ir.Type) (imported, bool) {
	i := int(t.GetExtern().GetIndex())
	if i < 0 || i >= len(g.Model.GetExterns()) {
		return imported{}, false
	}
	f, ok := g.externs[g.Model.GetExterns()[i]]
	return f, ok
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
			"%s is not checked: %s is a foreign type, whose values another module decides", emit.ConstraintText(c), d.GetMeta().GetName())
	}
}

// packageModule is the module a package generates without a `module`
// directive: its segments with each first letter upper-cased.
func packageModule(pkg string) string {
	segs := strings.Split(pkg, ".")
	for i, s := range segs {
		segs[i] = upperFirst(s)
	}
	return strings.Join(segs, ".")
}

// moduleName reports whether Haskell accepts a module name.
func moduleName(name string) bool {
	if name == "" {
		return false
	}
	for seg := range strings.SplitSeq(name, ".") {
		if !conid.MatchString(seg) {
			return false
		}
	}
	return true
}
