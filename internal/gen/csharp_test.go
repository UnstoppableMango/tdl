package gen_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/unstoppablemango/tdl/internal/analysis"
	"github.com/unstoppablemango/tdl/internal/gen"
	"github.com/unstoppablemango/tdl/internal/sema"
	"github.com/unstoppablemango/tdl/parser"
	"github.com/unstoppablemango/tdl/plugin"
)

// The C# backend is a .NET plugin, whose tests cannot lower TDL. Its
// requests are recorded here: testdata/csharp/cases holds the models its
// unit tests read, and testdata/csharp/corpus the conformance corpus and
// the smoke model, each with the files it generates.

const csharpTarget = "csharp"

// csharpRequest lowers a file and builds the request tdl sends the C#
// backend for it, whether or not the file has a csharp target block.
func csharpRequest(t *testing.T, path string) *plugin.Request {
	t.Helper()
	src, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	file, err := parser.Parse(filepath.ToSlash(path), strings.NewReader(string(src)))
	if err != nil {
		t.Fatalf("parsing %s: %v", path, err)
	}
	model, diags := sema.Lower(file, sema.WithLoader(sema.FSLoader{}))
	if len(diags) > 0 {
		t.Fatalf("lowering %s: %v", path, diags.Error())
	}
	req, _ := analysis.Request(model, plugin.Description{Name: csharpTarget}, csharpTarget, "")
	return req
}

// TestCSharpCases records a request for every file under
// testdata/csharp/cases, beside it.
func TestCSharpCases(t *testing.T) {
	sources, err := filepath.Glob("../../testdata/csharp/cases/*/*.tdl")
	if err != nil || len(sources) == 0 {
		t.Fatalf("no cases found: %v", err)
	}
	for _, src := range sources {
		t.Run(strings.TrimPrefix(src, "../../testdata/csharp/cases/"), func(t *testing.T) {
			checkBinary(t, strings.TrimSuffix(src, ".tdl")+".request.binpb", csharpRequest(t, src))
		})
	}
}

// TestCSharpCorpus records a request for every conformance case and the
// smoke model, and compares what tdl-gen-csharp generates for each with the
// files beside it when the plugin is on PATH. -record rewrites both.
func TestCSharpCorpus(t *testing.T) {
	cases, err := filepath.Glob("../../testdata/conformance/*")
	if err != nil {
		t.Fatal(err)
	}
	cases = slices.DeleteFunc(cases, func(dir string) bool {
		_, err := os.Stat(filepath.Join(dir, "pending"))
		return err == nil
	})
	cases = append(cases, "../../testdata/gen/smoke")

	backend, err := gen.Resolve(csharpTarget)
	if err != nil {
		backend = nil
	}

	want := map[string]bool{}
	for _, dir := range cases {
		name := filepath.Base(dir)
		want[name] = true
		t.Run(name, func(t *testing.T) {
			out := filepath.Join("../../testdata/csharp/corpus", name)
			if *record {
				if err := os.MkdirAll(out, 0o755); err != nil {
					t.Fatal(err)
				}
			}
			req := csharpRequest(t, filepath.Join(dir, "source.tdl"))
			checkBinary(t, filepath.Join(out, "request.binpb"), req)
			if backend == nil {
				t.Skip("tdl-gen-csharp is not on PATH")
			}
			resp, err := backend.Generate(context.Background(), req)
			if err != nil {
				t.Fatalf("generate: %v", err)
			}
			for _, d := range resp.GetDiagnostics() {
				if d.GetSeverity() == plugin.Severity_SEVERITY_ERROR {
					t.Errorf("error: %v", d)
				}
			}
			checkCorpusFiles(t, filepath.Join(out, "out"), resp.GetFiles())
		})
	}

	dirs, err := os.ReadDir("../../testdata/csharp/corpus")
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range dirs {
		if !want[d.Name()] {
			t.Errorf("testdata/csharp/corpus/%s is no corpus case; delete it", d.Name())
		}
	}
}

// checkCorpusFiles compares generated files with those in dir, or writes
// them there under -record.
func checkCorpusFiles(t *testing.T, dir string, files []*plugin.File) {
	t.Helper()
	if *record {
		if err := os.RemoveAll(dir); err != nil {
			t.Fatal(err)
		}
		for _, f := range files {
			path := filepath.Join(dir, filepath.FromSlash(f.GetPath()))
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, f.GetContent(), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		return
	}

	var got []string
	for _, f := range files {
		got = append(got, f.GetPath())
		want, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(f.GetPath())))
		if err != nil {
			t.Errorf("%s: %v (run `go test ./internal/gen -record`)", f.GetPath(), err)
			continue
		}
		if string(want) != string(f.GetContent()) {
			t.Errorf("%s is out of date (run `go test ./internal/gen -record`):\n%s", f.GetPath(), f.GetContent())
		}
	}
	var have []string
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, _ := filepath.Rel(dir, path)
		have = append(have, filepath.ToSlash(rel))
		return nil
	})
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		t.Fatal(err)
	}
	slices.Sort(got)
	if !slices.Equal(got, have) {
		t.Errorf("files = %v, want %v", got, have)
	}
}
