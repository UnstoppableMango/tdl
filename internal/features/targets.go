package features

import (
	"errors"

	"github.com/unstoppablemango/tdl/backend/smithy"
	"github.com/unstoppablemango/tdl/backend/typescript"
	"github.com/unstoppablemango/tdl/internal/gen"
)

// Corpus is testdata/features, relative to the module root.
const Corpus = "testdata/features"

// Targets is every built-in backend but debug, which describes a model
// rather than generating one. A backend whose import runs a tool needs it
// on PATH, and short skips running it at all.
func Targets(short bool) []Target {
	needs := map[string]func() error{
		smithy.Name: func() error {
			_, err := smithy.CLI()
			return err
		},
		typescript.Name: func() error {
			_, _, err := typescript.Compiler()
			return err
		},
	}
	var out []Target
	for _, name := range gen.BuiltinNames() {
		if name == "debug" {
			continue
		}
		b, _ := gen.Builtin(name)
		t := Target{Name: name, Backend: b, Needs: needs[name]}
		if short && t.Needs != nil {
			t.Needs = func() error { return errors.New("short") }
		}
		out = append(out, t)
	}
	return out
}
