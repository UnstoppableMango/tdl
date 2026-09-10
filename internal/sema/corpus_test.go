package sema

import (
	"flag"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	"github.com/unstoppablemango/tdl/ir"
	"github.com/unstoppablemango/tdl/parser"
)

// update rewrites the ir.golden files instead of checking them:
//
//	go test ./internal/sema -update
var update = flag.Bool("update", false, "rewrite ir.golden files")

// TestCorpusLowers walks the conformance corpus and checks that it lowers
// clean.
//
// It used to carry a list of diagnostics lowering was still expected to
// produce, each naming the phase that would stop producing it. Units were
// the last entry, so the list is gone and this is the plain assertion it
// was always going to become.
func TestCorpusLowers(t *testing.T) {
	dirs, err := filepath.Glob("../../testdata/conformance/*")
	if err != nil || len(dirs) == 0 {
		t.Fatalf("no corpus found: %v", err)
	}
	sort.Strings(dirs)

	for _, dir := range dirs {
		t.Run(filepath.Base(dir), func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join(dir, "source.tdl"))
			if err != nil {
				t.Fatalf("reading source.tdl: %v", err)
			}
			file, err := parser.Parse("source.tdl", strings.NewReader(string(data)))
			if err != nil {
				t.Fatalf("unexpected parse error: %v", err)
			}

			model, diags := Lower(file, WithLoader(caseLoader(t, dir)))
			if len(diags) > 0 {
				// Lower says a model that produced diagnostics is
				// incomplete, so the golden and the round-trip below would
				// be reporting on something nobody claimed was right.
				for _, d := range diags {
					t.Errorf("unexpected diagnostic: %s", d.Error())
				}
				return
			}

			checkGolden(t, filepath.Join(dir, "ir.golden"), ir.Dump(model))

			// What a plugin receives has to survive the trip.
			data, merr := protojson.Marshal(model)
			if merr != nil {
				t.Fatalf("marshalling: %v", merr)
			}
			var back ir.Model
			if err := protojson.Unmarshal(data, &back); err != nil {
				t.Fatalf("unmarshalling: %v", err)
			}
			if !proto.Equal(model, &back) {
				t.Error("the model did not survive a JSON round trip")
			}
		})
	}
}

// caseLoader serves the other .tdl files in a case directory by their base
// name, so the corpus can exercise imports while the goldens stay free of
// machine-specific paths.
func caseLoader(t *testing.T, dir string) MapLoader {
	t.Helper()

	paths, err := filepath.Glob(filepath.Join(dir, "*.tdl"))
	if err != nil {
		t.Fatalf("globbing %s: %v", dir, err)
	}

	m := MapLoader{}
	for _, p := range paths {
		if filepath.Base(p) == "source.tdl" {
			continue
		}
		src, err := os.ReadFile(p)
		if err != nil {
			t.Fatalf("reading %s: %v", p, err)
		}
		m[filepath.Base(p)] = string(src)
	}
	return m
}

// checkGolden compares got against the file at path, or rewrites it under
// -update.
func checkGolden(t *testing.T, path, got string) {
	t.Helper()

	if *update {
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatalf("writing %s: %v", path, err)
		}
		return
	}

	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v (run `go test ./internal/sema -update`)", path, err)
	}
	if got != string(want) {
		t.Errorf("%s is out of date:\n--- got ---\n%s\n--- want ---\n%s", path, got, want)
	}
}

// TestPreludeLowers checks the prelude itself, which is the one file that
// declares everything it uses.
func TestPreludeLowers(t *testing.T) {
	data, err := os.ReadFile("../../prelude/std.tdl")
	if err != nil {
		t.Fatalf("reading the prelude: %v", err)
	}
	file, err := parser.Parse("std.tdl", strings.NewReader(string(data)))
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}

	_, diags := Lower(file)
	for _, d := range diags {
		t.Errorf("unexpected diagnostic: %s", d.Error())
	}
}
