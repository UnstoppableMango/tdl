// Package debug is a backend that describes the model it was given.
//
// Its output is not useful, and that is the point. It exercises every part
// of the protocol, in process and over a subprocess, without anyone first
// having to agree what generated Go should look like. Real backends are
// each their own plan; this one exists to keep the protocol honest.
package debug

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/unstoppablemango/tdl/ir"
	"github.com/unstoppablemango/tdl/plugin"
)

// Name is what this backend is called, in a target block and as
// tdl-gen-debug on PATH.
const Name = "debug"

// Backend implements [plugin.Backend].
type Backend struct{}

func (Backend) Describe() plugin.Description {
	return plugin.Description{
		Name:    Name,
		Version: "0.1.0",
		Directives: []*plugin.DirectiveSpec{
			// `note("...")` gives a target block something to say that this
			// backend can echo, so directive plumbing has a user.
			{
				Name:     "note",
				MinArgs:  1,
				MaxArgs:  1,
				ArgKinds: []ir.LiteralKind{ir.LiteralKind_LITERAL_KIND_STRING},
			},
		},
	}
}

// Generate writes one file describing the model.
func (Backend) Generate(_ context.Context, req *plugin.Request) (*plugin.Response, error) {
	model := req.GetModel()

	var b strings.Builder
	fmt.Fprintf(&b, "package %s\n", model.GetPackage())
	fmt.Fprintf(&b, "target %s\n", req.GetTarget())
	fmt.Fprintf(&b, "\n")

	// The prelude is merged into the declaration table, so a model whose
	// source declares two things arrives with twenty-one declarations. A
	// backend that emits per declaration has to decide what is the user's;
	// this one splits by which file a declaration came from.
	own, borrowed := partition(model)
	fmt.Fprintf(&b, "declarations: %d own, %d from the prelude\n", len(own), len(borrowed))
	fmt.Fprintf(&b, "types: %d\n", len(model.GetTypes()))
	fmt.Fprintf(&b, "externs: %d\n", len(model.GetExterns()))
	fmt.Fprintf(&b, "instances: %d\n", len(model.GetInstances()))

	for _, d := range own {
		fmt.Fprintf(&b, "\n%s\n", d.GetMeta().GetName())
		for _, dir := range plugin.Directives(req.GetTarget(), d.GetDirectives()) {
			fmt.Fprintf(&b, "  directive %s\n", dir.GetName())
		}
		for _, f := range d.Fields() {
			fmt.Fprintf(&b, "  field %s: %s\n", f.GetMeta().GetName(), f.GetType().GetName())
		}
	}

	return &plugin.Response{
		Files: []*plugin.File{{
			Path:    "model.txt",
			Content: []byte(b.String()),
		}},
		Diagnostics: notes(own),
	}, nil
}

// notes reports something about the model, so the diagnostic path has a
// user. A backend says what it cannot handle here rather than returning an
// error, because this reaches the user with a position attached.
func notes(own []*ir.Decl) []*plugin.Diagnostic {
	var diags []*plugin.Diagnostic
	for _, d := range own {
		if len(d.Fields()) > 0 || d.GetEnumeration() != nil {
			continue
		}
		diags = append(diags, &plugin.Diagnostic{
			Severity: plugin.Severity_SEVERITY_WARNING,
			Message:  d.GetMeta().GetName() + " has no fields, so there is nothing to generate for it",
			Position: d.GetMeta().GetPosition(),
		})
	}
	return diags
}

// partition splits declarations by whether they came from the file being
// generated or from the prelude merged into it.
func partition(model *ir.Model) (own, borrowed []*ir.Decl) {
	counts := map[string]int{}
	for _, d := range model.GetDecls() {
		counts[d.GetMeta().GetPosition().GetFilename()]++
	}

	// The file with the most declarations that is not the one the model
	// names is the prelude. Rather than guess, take the model's own package
	// file as whatever is not the prelude's.
	prelude := ""
	names := make([]string, 0, len(counts))
	for name := range counts {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if strings.HasSuffix(name, "std.tdl") {
			prelude = name
		}
	}

	for _, d := range model.GetDecls() {
		if d.GetMeta().GetPosition().GetFilename() == prelude {
			borrowed = append(borrowed, d)
			continue
		}
		own = append(own, d)
	}
	return own, borrowed
}
