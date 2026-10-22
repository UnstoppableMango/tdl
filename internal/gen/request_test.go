package gen_test

import (
	"context"
	"slices"
	"testing"

	"google.golang.org/protobuf/proto"

	"github.com/unstoppablemango/tdl/internal/analysis"
	"github.com/unstoppablemango/tdl/internal/gen"
	"github.com/unstoppablemango/tdl/internal/host"
	"github.com/unstoppablemango/tdl/ir"
	"github.com/unstoppablemango/tdl/plugin"
)

// A backend that numbers writes the same files from the numbers tdl gives
// it as from the numbers it would assign itself.
func TestHostNumbersAsTheBackendWould(t *testing.T) {
	numbered := 0
	for _, name := range gen.BuiltinNames() {
		backend, _ := gen.Builtin(name)
		if backend.Describe().Numbering == nil {
			continue
		}
		numbered++
		for path, model := range corpora(t) {
			req, problems := analysis.Request(model, backend.Describe(), name, "out")
			if host.Fatal(problems) {
				t.Errorf("%s: %s: %v", path, name, problems)
				continue
			}
			fromHost, err := backend.Generate(context.Background(), req)
			if err != nil {
				t.Fatalf("%s: %s: %v", path, name, err)
			}
			own := proto.CloneOf(req)
			own.Analysis = nil
			fromBackend, err := backend.Generate(context.Background(), own)
			if err != nil {
				t.Fatalf("%s: %s: %v", path, name, err)
			}
			if !proto.Equal(fromHost, fromBackend) {
				t.Errorf("%s: %s writes different files from tdl's numbers", path, name)
			}
		}
	}
	if numbered == 0 {
		t.Fatal("no built-in backend numbers")
	}
}

// A backend writing roundtrip annotations writes the same files from the
// TDL tdl sends as from the TDL it would print itself.
func TestHostCarriesSourceAsTheBackendWould(t *testing.T) {
	carrying := 0
	for _, name := range gen.BuiltinNames() {
		backend, _ := gen.Builtin(name)
		if !slices.ContainsFunc(backend.Describe().Directives, func(d *plugin.DirectiveSpec) bool { return d.GetName() == "roundtrip" }) {
			continue
		}
		carrying++
		for path, model := range corpora(t) {
			req, problems := analysis.Request(withRoundtrip(model, name), backend.Describe(), name, "out")
			if host.Fatal(problems) {
				continue
			}
			if req.GetAnalysis().GetSource() == nil {
				t.Fatalf("%s: %s: the request carries no source under roundtrip", path, name)
			}
			fromHost, err := backend.Generate(context.Background(), req)
			if err != nil {
				t.Fatalf("%s: %s: %v", path, name, err)
			}
			own := proto.CloneOf(req)
			own.Analysis = nil
			fromBackend, err := backend.Generate(context.Background(), own)
			if err != nil {
				t.Fatalf("%s: %s: %v", path, name, err)
			}
			if !proto.Equal(fromHost, fromBackend) {
				t.Errorf("%s: %s writes different files from tdl's source", path, name)
			}
		}
	}
	if carrying == 0 {
		t.Fatal("no built-in backend writes roundtrip annotations")
	}
}

// withRoundtrip adds the roundtrip directive to target's block.
func withRoundtrip(model *ir.Model, target string) *ir.Model {
	out := proto.CloneOf(model)
	d := &ir.Directive{Name: "roundtrip", Target: target}
	for _, b := range out.GetTargets() {
		if b.GetMeta().GetName() == target {
			b.Directives = append(b.Directives, d)
			return out
		}
	}
	out.Targets = append(out.Targets, &ir.TargetBlock{
		Meta:       &ir.Meta{Name: target, Order: int32(len(out.GetTargets()))},
		ForPackage: out.GetPackage(),
		Directives: []*ir.Directive{d},
	})
	return out
}

// No corpus holds a constraint tdl refuses.
func TestCorporaHaveNoProblems(t *testing.T) {
	for path, model := range corpora(t) {
		for _, block := range model.GetTargets() {
			name := block.GetMeta().GetName()
			desc := plugin.Description{Name: name}
			if backend, ok := gen.Builtin(name); ok {
				desc = backend.Describe()
			}
			if _, problems := analysis.Request(model, desc, name, "out"); len(problems) > 0 {
				t.Errorf("%s: %s: %v", path, name, problems)
			}
		}
	}
}
