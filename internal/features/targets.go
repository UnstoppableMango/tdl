package features

import (
	"errors"
	"fmt"
	"path"
	"path/filepath"
	"reflect"

	"github.com/unstoppablemango/tdl/backend/smithy"
	"github.com/unstoppablemango/tdl/backend/typescript"
	"github.com/unstoppablemango/tdl/internal/gen"
)

// Corpus is testdata/features, relative to the module root.
const Corpus = "testdata/features"

// HandFile is the file beside a backend's source holding its hand-written
// cells.
const HandFile = "features.toml"

// plugins are the backends this repository serves as plugins rather than
// compiling into tdl, with the directory of their source. One is measured
// through tdl-gen-<name> on PATH.
var plugins = map[string]string{
	"csharp":  "dotnet/Tdl.Gen.CSharp",
	"mermaid": "js/tdl-gen-mermaid",
}

// Targets is every built-in backend but debug, which describes a model
// rather than generating one, and every plugin in plugins. A backend whose
// import runs a tool needs it on PATH, and a plugin missing from PATH is
// not measured; short measures neither, leaving their cells unmeasured.
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
		dir := filepath.Join("backend", path.Base(reflect.TypeOf(b).PkgPath()))
		t := Target{Name: name, Backend: b, Needs: needs[name], Hand: filepath.Join(dir, HandFile)}
		if short && t.Needs != nil {
			t.Needs = func() error { return errors.New("short") }
		}
		out = append(out, t)
	}
	for _, name := range sortedKeys(plugins) {
		t := Target{Name: name, Hand: filepath.Join(plugins[name], HandFile)}
		if b, err := gen.Find(name); err == nil && !short {
			t.Backend = b
		}
		out = append(out, t)
	}
	return out
}

// missing says why t cannot be measured whole here: a plugin missing from
// PATH, or a tool its import runs. Nil means nothing is missing.
func (t Target) missing() error {
	if t.Backend == nil {
		return fmt.Errorf("%s%s is not on PATH", gen.CommandPrefix, t.Name)
	}
	if t.Needs != nil {
		return t.Needs()
	}
	return nil
}
