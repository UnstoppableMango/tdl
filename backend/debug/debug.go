// Package debug is a backend that describes the model it was given. It
// exercises the plugin protocol, in process and over a subprocess, without
// generating code.
package debug

import (
	"context"
	"fmt"
	"strings"

	"github.com/unstoppablemango/tdl/ir"
	"github.com/unstoppablemango/tdl/plugin"
	"github.com/unstoppablemango/tdl/prelude"
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
		// Requests share no state.
		Reuse: true,
		Directives: []*plugin.DirectiveSpec{
			// `note("...")` exists to exercise directive plumbing.
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

// notes warns about each fieldless declaration, to exercise the diagnostic
// path.
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

// partition splits declarations into the model's and the prelude's. The
// prelude's name is matched whole, so a model file `mystd.tdl` is the
// model's.
func partition(model *ir.Model) (own, borrowed []*ir.Decl) {
	for _, d := range model.GetDecls() {
		if d.GetMeta().GetPosition().GetFilename() == prelude.Name {
			borrowed = append(borrowed, d)
			continue
		}
		own = append(own, d)
	}
	return own, borrowed
}
