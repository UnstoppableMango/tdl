package host

import (
	"slices"

	"github.com/unstoppablemango/tdl/ir"
	"github.com/unstoppablemango/tdl/plugin"
)

// PostDirective names, in a target block, the commands tdl.toml's [post]
// table maps to command lines, which tdl runs over every file the target
// returns before anything is compared or written.
const PostDirective = "post"

// CheckDirective names, in a target block, the commands tdl.toml's [check]
// table maps to command lines, which tdl runs over a staged copy of the
// target's post-processed output before writing it.
const CheckDirective = "check"

// hostSpecs are the directives tdl reads itself beside out, which every
// backend accepts and none need declare. Each takes one or more names, as
// strings, in the target block, and may be repeated, its names following
// the earlier entry's.
var hostSpecs = map[string]*plugin.DirectiveSpec{
	PostDirective:  nameList(PostDirective),
	CheckDirective: nameList(CheckDirective),
}

func nameList(name string) *plugin.DirectiveSpec {
	return &plugin.DirectiveSpec{
		Name:       name,
		MinArgs:    1,
		MaxArgs:    -1,
		Placements: plugin.On(plugin.OnBlock),
		Repeatable: true,
	}
}

// Directives lists the names of the directives tdl reads itself and holds
// to its own spec, out aside.
func Directives() []string {
	names := make([]string, 0, len(hostSpecs))
	for name := range hostSpecs {
		names = append(names, name)
	}
	slices.Sort(names)
	return names
}

// withHost puts the host's directives into specs, in place of any a
// backend declares under the same names.
func withHost(specs map[string]*plugin.DirectiveSpec) {
	for name, spec := range hostSpecs {
		specs[name] = spec
	}
}

// checkNames reports a host directive's argument that is not a string; the
// spec's arg_kinds constrain only the positions they list.
func checkNames(d *ir.Directive) []*plugin.Diagnostic {
	if _, ok := hostSpecs[d.GetName()]; !ok {
		return nil
	}
	var problems []*plugin.Diagnostic
	for i, arg := range d.GetArgs() {
		if arg.GetKind() != ir.LiteralKind_LITERAL_KIND_STRING {
			problems = append(problems, problem(arg.GetPosition(), plugin.Severity_SEVERITY_ERROR,
				"%s argument %d is %s, want %s", d.GetName(), i+1, ir.KindName(arg.GetKind()), ir.KindName(ir.LiteralKind_LITERAL_KIND_STRING)))
		}
	}
	return problems
}

// Names reads every name a target block's directive gives, in the order
// written, across repeated entries.
func Names(block *ir.TargetBlock, directive string) []string {
	var names []string
	for _, d := range block.GetDirectives() {
		if d.GetName() != directive {
			continue
		}
		for _, arg := range d.GetArgs() {
			names = append(names, arg.GetText())
		}
	}
	return names
}
