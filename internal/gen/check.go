package gen

import (
	"fmt"

	"github.com/unstoppablemango/tdl/ir"
	"github.com/unstoppablemango/tdl/plugin"
)

// compilerOwned are directives tdl reads itself. A backend never sees
// them and is not asked to declare them.
var compilerOwned = map[string]bool{
	"out": true,
}

// Problem is something wrong with a target block, found before any
// backend runs.
type Problem struct {
	Position *ir.Position
	Message  string
	Warning  bool
}

func (p Problem) String() string {
	kind := "error"
	if p.Warning {
		kind = "warning"
	}
	return fmt.Sprintf("%s:%d:%d: %s: %s",
		p.Position.GetFilename(), p.Position.GetLine(), p.Position.GetColumn(),
		kind, p.Message)
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
func CheckDirectives(target string, model *ir.Model, desc plugin.Description) []Problem {
	specs := map[string]*plugin.DirectiveSpec{}
	for _, s := range desc.Directives {
		specs[s.GetName()] = s
	}

	var problems []Problem
	for _, d := range directivesFor(target, model) {
		if compilerOwned[d.GetName()] {
			continue
		}
		spec, declared := specs[d.GetName()]
		if !declared {
			problems = append(problems, Problem{
				Position: d.GetPosition(),
				Message:  fmt.Sprintf("%s does not declare the directive %s", target, d.GetName()),
				Warning:  true,
			})
			continue
		}
		problems = append(problems, checkOne(target, d, spec)...)
	}
	return problems
}

func checkOne(target string, d *ir.Directive, spec *plugin.DirectiveSpec) []Problem {
	var problems []Problem
	n := int32(len(d.GetArgs()))

	switch {
	case n < spec.GetMinArgs():
		problems = append(problems, Problem{
			Position: d.GetPosition(),
			Message: fmt.Sprintf("%s takes at least %d argument(s), got %d",
				d.GetName(), spec.GetMinArgs(), n),
		})
	case spec.GetMaxArgs() >= 0 && n > spec.GetMaxArgs():
		problems = append(problems, Problem{
			Position: d.GetPosition(),
			Message: fmt.Sprintf("%s takes at most %d argument(s), got %d",
				d.GetName(), spec.GetMaxArgs(), n),
		})
	}

	// arg_kinds constrains by position, and a shorter list constrains only
	// what it covers.
	for i, want := range spec.GetArgKinds() {
		if i >= len(d.GetArgs()) {
			break
		}
		if got := d.GetArgs()[i].GetKind(); got != want {
			problems = append(problems, Problem{
				Position: d.GetArgs()[i].GetPosition(),
				Message: fmt.Sprintf("%s argument %d is %s, want %s",
					d.GetName(), i+1, kindName(got), kindName(want)),
			})
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

func kindName(k ir.LiteralKind) string {
	switch k {
	case ir.LiteralKind_LITERAL_KIND_STRING:
		return "a string"
	case ir.LiteralKind_LITERAL_KIND_INT:
		return "an integer"
	case ir.LiteralKind_LITERAL_KIND_FLOAT:
		return "a float"
	case ir.LiteralKind_LITERAL_KIND_BOOL:
		return "a boolean"
	case ir.LiteralKind_LITERAL_KIND_NAME:
		return "a name"
	case ir.LiteralKind_LITERAL_KIND_REGEX:
		return "a regex"
	case ir.LiteralKind_LITERAL_KIND_LIST:
		return "a list"
	case ir.LiteralKind_LITERAL_KIND_RANGE:
		return "a range"
	}
	return "unspecified"
}
