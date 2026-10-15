package gen

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

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
// A declared directive with the wrong number or kind of arguments, written
// where its spec does not place it, or given a value its spec does not
// list is an error, reported before anything is generated. An undeclared
// directive is a warning and is passed through anyway, since a backend may
// handle more than it advertises.
func CheckDirectives(target string, model *ir.Model, desc plugin.Description) []*plugin.Diagnostic {
	specs := map[string]*plugin.DirectiveSpec{}
	for _, s := range desc.Directives {
		specs[s.GetName()] = s
	}

	var problems []*plugin.Diagnostic
	for _, n := range nodeDirectives(target, model) {
		problems = append(problems, checkEach(target, n, specs)...)
		problems = append(problems, checkTies(n.directives, specs)...)
	}
	return problems
}

// node is the directives a target resolves onto one node, and the kind of
// node it is.
type node struct {
	placement  plugin.Placement
	directives []*ir.Directive
}

// placementNames say where a directive was written, for a diagnostic.
var placementNames = map[plugin.Placement]string{
	plugin.Placement_PLACEMENT_BLOCK:   "the target block",
	plugin.Placement_PLACEMENT_DECL:    "a declaration",
	plugin.Placement_PLACEMENT_FIELD:   "a field",
	plugin.Placement_PLACEMENT_VARIANT: "a variant",
	plugin.Placement_PLACEMENT_EXTERN:  "an extern",
}

// placed reports whether spec accepts a directive on a node of kind at. A
// spec placing nothing accepts it anywhere, and one accepting a
// declaration accepts an extern, which is a declaration another package
// owns.
func placed(spec *plugin.DirectiveSpec, at plugin.Placement) bool {
	ps := spec.GetPlacements()
	if len(ps) == 0 || slices.Contains(ps, at) {
		return true
	}
	return at == plugin.Placement_PLACEMENT_EXTERN && slices.Contains(ps, plugin.Placement_PLACEMENT_DECL)
}

// where lists the placements a spec accepts, for a diagnostic.
func where(spec *plugin.DirectiveSpec) string {
	var names []string
	for _, p := range spec.GetPlacements() {
		names = append(names, placementNames[p])
	}
	return orList(names)
}

// orList joins items as "a", "a or b", or "a, b, or c".
func orList(items []string) string {
	switch len(items) {
	case 0:
		return ""
	case 1:
		return items[0]
	case 2:
		return items[0] + " or " + items[1]
	}
	return strings.Join(items[:len(items)-1], ", ") + ", or " + items[len(items)-1]
}

func checkEach(target string, n node, specs map[string]*plugin.DirectiveSpec) []*plugin.Diagnostic {
	ds := n.directives
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
		if !placed(spec, n.placement) {
			problems = append(problems, problem(d.GetPosition(), plugin.Severity_SEVERITY_ERROR,
				"%s is written on %s, and %s reads it only on %s", d.GetName(), placementNames[n.placement], target, where(spec)))
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

// nodeDirectives returns a target's directives on each of its blocks,
// declarations, fields, variants, and externs, one slice per node.
func nodeDirectives(target string, model *ir.Model) []node {
	var nodes []node

	add := func(at plugin.Placement, ds []*ir.Directive) {
		nodes = append(nodes, node{at, plugin.Directives(target, ds)})
	}
	for _, block := range model.GetTargets() {
		if block.GetMeta().GetName() == target {
			add(plugin.Placement_PLACEMENT_BLOCK, block.GetDirectives())
		}
	}
	for _, decl := range model.GetDecls() {
		add(plugin.Placement_PLACEMENT_DECL, decl.GetDirectives())
		for _, f := range decl.Fields() {
			add(plugin.Placement_PLACEMENT_FIELD, f.GetDirectives())
		}
		for _, v := range decl.GetEnumeration().GetVariants() {
			add(plugin.Placement_PLACEMENT_VARIANT, v.GetDirectives())
			for _, f := range v.GetFields() {
				add(plugin.Placement_PLACEMENT_FIELD, f.GetDirectives())
			}
		}
	}
	for _, e := range model.GetExterns() {
		add(plugin.Placement_PLACEMENT_EXTERN, e.GetDirectives())
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

	if vs := spec.GetValues(); len(vs) > 0 && len(d.GetArgs()) > 0 {
		if got := d.GetArgs()[0].GetText(); !slices.Contains(vs, got) {
			quoted := make([]string, len(vs))
			for i, v := range vs {
				quoted[i] = strconv.Quote(v)
			}
			problems = append(problems, problem(d.GetArgs()[0].GetPosition(), plugin.Severity_SEVERITY_ERROR,
				"%s does not take %q; use %s", d.GetName(), got, orList(quoted)))
		}
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
