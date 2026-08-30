package parser_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/unstoppablemango/tdl/parser"
)

// TestConformanceCorpusParses walks testdata/conformance and checks that
// every source.tdl parses without error. This corpus is plain text data,
// not Go code, so a future non-Go TDL implementation can run the same
// check against the same files (see docs/spec.md).
func TestConformanceCorpusParses(t *testing.T) {
	for _, dir := range subdirs(t, "../testdata/conformance") {
		t.Run(filepath.Base(dir), func(t *testing.T) {
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

// TestInvalidCorpusFails walks testdata/invalid and checks that every
// source.tdl fails to parse with an error containing the text in the
// sibling error.golden file.
func TestInvalidCorpusFails(t *testing.T) {
	for _, dir := range subdirs(t, "../testdata/invalid") {
		t.Run(filepath.Base(dir), func(t *testing.T) {
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
