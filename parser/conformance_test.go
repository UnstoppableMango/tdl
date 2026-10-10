package parser_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/unstoppablemango/tdl/ast"
	"github.com/unstoppablemango/tdl/parser"
)

// TestConformanceCorpusParses checks every testdata/conformance source.tdl
// parses. A case directory holding a `pending` file is skipped.
func TestConformanceCorpusParses(t *testing.T) {
	for _, dir := range subdirs(t, "../testdata/conformance") {
		t.Run(filepath.Base(dir), func(t *testing.T) {
			skipPending(t, dir)
			data, err := os.ReadFile(filepath.Join(dir, "source.tdl"))
			if err != nil {
				t.Fatalf("reading source.tdl: %v", err)
			}
			if _, err := parser.Parse("source.tdl", strings.NewReader(string(data))); err != nil {
				t.Fatalf("unexpected parse error: %v", err)
			}
		})
	}
}

// TestInvalidCorpusFails checks every testdata/invalid source.tdl fails
// with an error containing its sibling error.golden.
func TestInvalidCorpusFails(t *testing.T) {
	for _, dir := range subdirs(t, "../testdata/invalid") {
		t.Run(filepath.Base(dir), func(t *testing.T) {
			skipPending(t, dir)
			data, err := os.ReadFile(filepath.Join(dir, "source.tdl"))
			if err != nil {
				t.Fatalf("reading source.tdl: %v", err)
			}
			want, err := os.ReadFile(filepath.Join(dir, "error.golden"))
			if err != nil {
				t.Fatalf("reading error.golden: %v", err)
			}

			_, err = parser.Parse("source.tdl", strings.NewReader(string(data)))
			if err == nil {
				t.Fatal("expected a parse error, got none")
			}
			if !strings.Contains(err.Error(), strings.TrimSpace(string(want))) {
				t.Fatalf("error %q does not contain expected substring %q", err.Error(), strings.TrimSpace(string(want)))
			}
		})
	}
}

// TestCorpusIsCanonical checks `tdl fmt` prints each stored .tdl file back
// byte for byte.
func TestCorpusIsCanonical(t *testing.T) {
	for _, dir := range subdirs(t, "../testdata/conformance") {
		t.Run(filepath.Base(dir), func(t *testing.T) {
			skipPending(t, dir)
			assertCanonical(t, filepath.Join(dir, "source.tdl"))
		})
	}

	// examples/ covers comments surviving a round trip on a real file.
	for _, dir := range []string{
		"../prelude", "../examples", "../testdata/gen/smoke",
		"../models/unist", "../models/mdast", "../models/hast",
	} {
		t.Run(filepath.Base(dir), func(t *testing.T) {
			matches, err := filepath.Glob(filepath.Join(dir, "*.tdl"))
			if err != nil {
				t.Fatalf("globbing %s: %v", dir, err)
			}
			if len(matches) == 0 {
				t.Fatalf("no sources found in %s", dir)
			}
			for _, path := range matches {
				t.Run(filepath.Base(path), func(t *testing.T) {
					assertCanonical(t, path)
				})
			}
		})
	}
}

func assertCanonical(t *testing.T, path string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}

	src := string(data)
	file, err := parser.Parse(path, strings.NewReader(src))
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}

	got, want := ast.Fprint(file), src
	if got == want {
		return
	}
	t.Errorf("%s is not canonical; run: tdl fmt -w %s\nfirst difference at %s",
		path, path, firstDiff(got, want))
}

// firstDiff describes the first line where got and want differ.
func firstDiff(got, want string) string {
	g, w := strings.Split(got, "\n"), strings.Split(want, "\n")
	for i := 0; i < len(g) && i < len(w); i++ {
		if g[i] != w[i] {
			return fmt.Sprintf("line %d:\n  got:  %q\n  want: %q", i+1, g[i], w[i])
		}
	}
	return fmt.Sprintf("end of file: got %d lines, want %d", len(g), len(w))
}

// skipPending skips a case holding a `pending` file, with its text as the
// reason.
func skipPending(t *testing.T, dir string) {
	t.Helper()
	reason, err := os.ReadFile(filepath.Join(dir, "pending"))
	if err != nil {
		return
	}
	t.Skip(strings.TrimSpace(string(reason)))
}

func subdirs(t *testing.T, root string) []string {
	t.Helper()
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatalf("reading %s: %v", root, err)
	}
	var dirs []string
	for _, e := range entries {
		if e.IsDir() {
			dirs = append(dirs, filepath.Join(root, e.Name()))
		}
	}
	return dirs
}
