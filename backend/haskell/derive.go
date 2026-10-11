package haskell

import (
	"slices"
	"strings"

	"github.com/unstoppablemango/tdl/backend/internal/emit"
	"github.com/unstoppablemango/tdl/ir"
	"github.com/unstoppablemango/tdl/plugin"
)

// derived is the classes `derive` directives add to a declaration: the
// target block's, which apply to every declaration, then its own, each
// once.
func (g *generator) derived(d *ir.Decl) []imported {
	var all []*ir.Directive
	for _, block := range g.Model.GetTargets() {
		if block.GetMeta().GetName() == g.Target {
			all = append(all, plugin.Directives(g.Target, block.GetDirectives())...)
		}
	}
	all = append(all, plugin.Directives(g.Target, d.GetDirectives())...)

	var out []imported
	for _, dir := range all {
		if dir.GetName() != "derive" {
			continue
		}
		args := dir.GetArgs()
		// A target block's directive is read for every declaration, and
		// warns once.
		warn := func(err error) {
			if !g.warned[dir] {
				g.warned[dir] = true
				g.Warn(err)
			}
		}
		if len(args) != 2 {
			warn(emit.Unsupported(dir.GetPosition(), "derive takes a module and a class name, and %d argument(s) were given", len(args)))
			continue
		}
		c := imported{module: args[0].GetText(), name: args[1].GetText()}
		switch {
		case !moduleName(c.module):
			warn(emit.Unsupported(dir.GetPosition(), "derive names %q, which is not a Haskell module name", c.module))
		case !conid.MatchString(c.name):
			warn(emit.Unsupported(dir.GetPosition(), "derive names %q, which is not a Haskell class name", c.name))
		case !slices.Contains(out, c):
			out = append(out, c)
		}
	}
	return out
}

// anyclass is the deriving clause for the classes `derive` adds, or "" for
// none.
func (g *generator) anyclass(classes []imported) string {
	if len(classes) == 0 {
		return ""
	}
	g.exts["DeriveAnyClass"] = true
	names := make([]string, len(classes))
	for i, c := range classes {
		names[i] = g.use(c)
	}
	return "  deriving anyclass (" + strings.Join(names, ", ") + ")\n"
}
