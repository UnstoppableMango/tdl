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

// deferred maps a diagnostic to the phase that will stop producing it.
// Anything a corpus case reports that is not on this list is a bug in
// lowering, not a gap in it.
//
// The list shrinks as docs/design/ir-plan.md is worked through, and the
// test tightens with it: when the last entry goes, this becomes a plain
// assertion that the corpus lowers clean.
var deferred = []struct{ prefix, phase string }{
	{"undefined: ", "phase 4, when the prelude is loaded"},
	{"imports are not resolved yet", "phase 5"},
	{"qualified name ", "phase 5"},
	{"unit arguments are not lowered yet", "deferred in ir.md"},
	{" is not lowered yet", "phases 6 and 8"},
}

func deferralFor(msg string) (string, bool) {
	for _, d := range deferred {
		if strings.HasPrefix(msg, d.prefix) || strings.Contains(msg, d.prefix) {
			return d.phase, true
		}
	}
	return "", false
}

// TestCorpusLowers walks the conformance corpus and checks that every
// diagnostic lowering produces is a known deferral.
//
// The corpus is the written-down target, so it runs ahead of the
// implementation on purpose. What this guards is that it runs ahead in
// exactly the ways the plan says and no others.
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

			model, diags := Lower(file)
			for _, d := range diags {
				if _, ok := deferralFor(d.Msg); !ok {
					t.Errorf("unexpected diagnostic: %s", d.Error())
				}
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
		if _, ok := deferralFor(d.Msg); !ok {
			t.Errorf("unexpected diagnostic: %s", d.Error())
		}
	}
}
