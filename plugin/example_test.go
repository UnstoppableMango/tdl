package plugin_test

import (
	"context"
	"fmt"
	"strings"

	"github.com/unstoppablemango/tdl/ir"
	"github.com/unstoppablemango/tdl/plugin"
)

// shout is a backend in full: it declares what it understands, and it
// turns a model into files. A real one differs only in what it writes.
type shout struct{}

func (shout) Describe() plugin.Description {
	return plugin.Description{
		Name:    "shout",
		Version: "1.0.0",
		Directives: []*plugin.DirectiveSpec{{
			Name:     "as",
			MinArgs:  1,
			MaxArgs:  1,
			ArgKinds: []ir.LiteralKind{ir.LiteralKind_LITERAL_KIND_STRING},
		}},
	}
}

func (shout) Generate(_ context.Context, req *plugin.Request) (*plugin.Response, error) {
	var b strings.Builder

	for _, decl := range req.GetModel().GetDecls() {
		name := decl.GetMeta().GetName()

		// A model carries directives for every target block in it. Filter,
		// or you will act on another backend's instructions.
		for _, d := range plugin.Directives(req.GetTarget(), decl.GetDirectives()) {
			if d.GetName() == "as" && len(d.GetArgs()) == 1 {
				name = d.GetArgs()[0].GetText()
			}
		}

		fmt.Fprintln(&b, strings.ToUpper(name))
	}

	return &plugin.Response{
		Files: []*plugin.File{{Path: "shouted.txt", Content: []byte(b.String())}},
	}, nil
}

// A plugin's main is Serve and nothing else.
func Example_backend() {
	model := &ir.Model{
		Decls: []*ir.Decl{
			{Meta: &ir.Meta{Name: "Order"}},
			{
				Meta:       &ir.Meta{Name: "Customer"},
				Directives: []*ir.Directive{{Name: "as", Target: "shout", Args: []*ir.Literal{{Text: "buyer"}}}},
			},
		},
	}

	resp, err := shout{}.Generate(context.Background(), &plugin.Request{
		Target: "shout",
		Model:  model,
	})
	if err != nil {
		panic(err)
	}
	fmt.Print(string(resp.GetFiles()[0].GetContent()))

	// Output:
	// ORDER
	// BUYER
}
