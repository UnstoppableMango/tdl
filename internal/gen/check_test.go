package gen_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/unstoppablemango/tdl/internal/gen"
	"github.com/unstoppablemango/tdl/internal/host"
	"github.com/unstoppablemango/tdl/internal/sema"
	"github.com/unstoppablemango/tdl/ir"
	"github.com/unstoppablemango/tdl/parser"
	"github.com/unstoppablemango/tdl/plugin"
)

// TestCorporaPlaceDirectives holds every built-in backend's placements to
// what the files in the repository write.
func TestCorporaPlaceDirectives(t *testing.T) {
	for path, model := range corpora(t) {
		for _, block := range model.GetTargets() {
			name := block.GetMeta().GetName()
			backend, ok := gen.Builtin(name)
			if !ok {
				continue
			}
			for _, p := range host.CheckDirectives(name, model, backend.Describe()) {
				if p.GetSeverity() == plugin.Severity_SEVERITY_ERROR {
					t.Errorf("%s: %s: %s", path, name, p.GetMessage())
				}
			}
		}
	}
}

// corpora lowers every file in the corpora and examples that lowers, by
// path.
func corpora(t *testing.T) map[string]*ir.Model {
	t.Helper()
	var paths []string
	for _, glob := range []string{
		"../../testdata/conformance/*/source.tdl",
		"../../testdata/gen/*/source.tdl",
		"../../testdata/roundtrip/*/*/*.tdl",
		"../../examples/*.tdl",
	} {
		found, err := filepath.Glob(glob)
		if err != nil {
			t.Fatal(err)
		}
		paths = append(paths, found...)
	}

	models := map[string]*ir.Model{}
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		file, err := parser.Parse(path, bytes.NewReader(data))
		if err != nil {
			continue // the invalid corpus and pending cases are checked elsewhere
		}
		model, diags := sema.Lower(file, sema.WithLoader(sema.FSLoader{}))
		if len(diags) > 0 {
			continue
		}
		models[path] = model
	}
	return models
}
