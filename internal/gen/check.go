package gen

import (
	"fmt"

	"github.com/unstoppablemango/tdl/ir"
	"github.com/unstoppablemango/tdl/plugin"
)

// outDirective is the one directive tdl reads itself: where a target
// block's output goes. A backend need not declare it.
const outDirective = "out"

// problem reports something wrong with a target block, found before any
// backend runs.
func problem(pos *ir.Position, severity plugin.Severity, format string, args ...any) *plugin.Diagnostic {
	return &plugin.Diagnostic{Severity: severity, Message: fmt.Sprintf(format, args...), Position: pos}
}

// CheckDirectives compares the directives a target block uses against what
// its backend says it understands.
//
// A declared directive with the wrong number or kind of arguments is an
// error, reported before anything is generated. An undeclared directive
// is a warning and is passed through anyway, since a backend may handle
// more than it advertises.
func CheckDirectives(target string, model *ir.Model, desc plugin.Description) []*plugin.Diagnostic {
	specs := map[string]*plugin.DirectiveSpec{}
	for _, s := range desc.Directives {
		specs[s.GetName()] = s
	}

	var problems []*plugin.Diagnostic
	for _, block := range model.GetTargets() {
		if block.GetMeta().GetName() == target {
			problems = append(problems, checkEach(target, block.GetDirectives(), specs)...)
		}
	}
	for _, ds := range nodeDirectives(target, model) {
		problems = append(problems, checkEach(target, ds, specs)...)
		problems = append(problems, checkTies(ds, specs)...)
	}
	return problems
}

func checkEach(target string, ds []*ir.Directive, specs map[string]*plugin.DirectiveSpec) []*plugin.Diagnostic {
	var problems []*plugin.Diagnostic
	for _, d := range ds {
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

// checkTies reports a name appearing twice on one node. Lowering keeps
// every entry at the winning specificity, so a repeat is a tie, allowed
// only for a directive declared repeatable.
func checkTies(ds []*ir.Directive, specs map[string]*plugin.DirectiveSpec) []*plugin.Diagnostic {
	var problems []*plugin.Diagnostic
	seen := map[string]bool{}
	for _, d := range ds {
		if seen[d.GetName()] && !specs[d.GetName()].GetRepeatable() {
			problems = append(problems, problem(d.GetPosition(), plugin.Severity_SEVERITY_ERROR,
				"two entries at the same specificity set %s; one of them has to go", d.GetName()))
		}
		seen[d.GetName()] = true
	}
	return problems
}

// nodeDirectives returns a target's directives on each declaration,
// field, variant, and extern, one slice per node.
func nodeDirectives(target string, model *ir.Model) [][]*ir.Directive {
	var nodes [][]*ir.Directive

	add := func(ds []*ir.Directive) {
		nodes = append(nodes, plugin.Directives(target, ds))
	}
	for _, decl := range model.GetDecls() {
		add(decl.GetDirectives())
		for _, f := range decl.Fields() {
			add(f.GetDirectives())
		}
		for _, v := range decl.GetEnumeration().GetVariants() {
			add(v.GetDirectives())
			for _, f := range v.GetFields() {
				add(f.GetDirectives())
			}
		}
	}
	for _, e := range model.GetExterns() {
		add(e.GetDirectives())
	}
	return nodes
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

	// arg_kinds constrains by position; a shorter list constrains only the
	// arguments it covers.
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
