package gen

import (
	"fmt"

	"github.com/unstoppablemango/tdl/ir"
	"github.com/unstoppablemango/tdl/plugin"
)

// outDirective is the one directive tdl reads itself: where a target
// block's output goes. A backend never sees it and is not asked to
// declare it.
const outDirective = "out"

// problem reports something wrong with a target block, found before any
// backend runs, in the shape a backend's own diagnostics take.
func problem(pos *ir.Position, severity plugin.Severity, format string, args ...any) *plugin.Diagnostic {
	return &plugin.Diagnostic{Severity: severity, Message: fmt.Sprintf(format, args...), Position: pos}
}

// CheckDirectives compares the directives a target block uses against what
// its backend says it understands.
//
// A declared directive used with the wrong number or kind of arguments is
// an error, reported with the position in the .tdl file, before anything
// is generated: the alternative is a backend discovering it half way
// through writing files.
//
// A directive the backend did not declare is a warning and is passed
// through anyway. Under-declaring is a plugin bug that should not break a
// working project, and a backend is free to handle more than it
// advertises. The warning still names it, so a typo stays visible.
func CheckDirectives(target string, model *ir.Model, desc plugin.Description) []*plugin.Diagnostic {
	specs := map[string]*plugin.DirectiveSpec{}
	for _, s := range desc.Directives {
		specs[s.GetName()] = s
	}

	var problems []*plugin.Diagnostic
	for _, d := range directivesFor(target, model) {
		if d.GetName() == outDirective {
			continue
		}
		spec, declared := specs[d.GetName()]
		if !declared {
			problems = append(problems, problem(d.GetPosition(), plugin.Severity_SEVERITY_WARNING,
				"%s does not declare the directive %s", target, d.GetName()))
			continue
		}
		problems = append(problems, checkOne(d, spec)...)
	}
	return problems
}

func checkOne(d *ir.Directive, spec *plugin.DirectiveSpec) []*plugin.Diagnostic {
	var problems []*plugin.Diagnostic
	n := int32(len(d.GetArgs()))

	switch {
	case n < spec.GetMinArgs():
		problems = append(problems, problem(d.GetPosition(), plugin.Severity_SEVERITY_ERROR,
			"%s takes at least %d argument(s), got %d", d.GetName(), spec.GetMinArgs(), n))
	case spec.GetMaxArgs() >= 0 && n > spec.GetMaxArgs():
		problems = append(problems, problem(d.GetPosition(), plugin.Severity_SEVERITY_ERROR,
			"%s takes at most %d argument(s), got %d", d.GetName(), spec.GetMaxArgs(), n))
	}

	// arg_kinds constrains by position, and a shorter list constrains only
	// what it covers.
	for i, want := range spec.GetArgKinds() {
		if i >= len(d.GetArgs()) {
			break
		}
		if got := d.GetArgs()[i].GetKind(); got != want {
			problems = append(problems, problem(d.GetArgs()[i].GetPosition(), plugin.Severity_SEVERITY_ERROR,
				"%s argument %d is %s, want %s", d.GetName(), i+1, ir.KindName(got), ir.KindName(want)))
		}
	}
	return problems
}

// directivesFor collects every directive belonging to a target, wherever
// it ended up attached.
func directivesFor(target string, model *ir.Model) []*ir.Directive {
	var all []*ir.Directive

	add := func(ds []*ir.Directive) {
		all = append(all, plugin.Directives(target, ds)...)
	}
	for _, block := range model.GetTargets() {
		if block.GetMeta().GetName() == target {
			all = append(all, block.GetDirectives()...)
		}
	}
	for _, decl := range model.GetDecls() {
		add(decl.GetDirectives())
		for _, f := range decl.Fields() {
			add(f.GetDirectives())
		}
	}
	return all
}
