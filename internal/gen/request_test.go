package gen_test

import (
	"context"
	"testing"

	"google.golang.org/protobuf/proto"

	"github.com/unstoppablemango/tdl/internal/analysis"
	"github.com/unstoppablemango/tdl/internal/gen"
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
			if gen.Fatal(problems) {
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
