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

// TestConformanceCorpusParses walks testdata/conformance and checks that
// every source.tdl parses without error. This corpus is plain text data,
// not Go code, so a future non-Go TDL implementation can run the same
// check against the same files (see docs/spec.md).
//
// A case directory containing a `pending` file describes a construct the
// parser cannot read yet and is skipped. The phase that implements the
// construct deletes the marker; see docs/design/parser-plan.md.
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

// TestInvalidCorpusFails walks testdata/invalid and checks that every
// source.tdl fails to parse with an error containing the text in the
// sibling error.golden file.
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

// TestCorpusIsCanonical checks that every .tdl file the repository holds to
// canonical form is stored that way: `tdl fmt` over it must print it back
// byte for byte.
//
// docs/spec.md states the property and AGENTS.md names the files, and
// nothing else asserts it. It is also what says a change to the printer
// left existing output alone.
func TestCorpusIsCanonical(t *testing.T) {
	for _, dir := range subdirs(t, "../testdata/conformance") {
		t.Run(filepath.Base(dir), func(t *testing.T) {
			skipPending(t, dir)
			assertCanonical(t, filepath.Join(dir, "source.tdl"))
		})
	}

	// examples/ carries the explanatory comments the corpus does not, so
	// it is what says a comment survives a round trip through the
	// formatter on a real file rather than only on a fixture.
	for _, dir := range []string{"../prelude", "../examples"} {
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

// assertCanonical parses path and compares ast.Fprint against the bytes on
// disk, reporting the first line that differs.
func assertCanonical(t *testing.T, path string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}

	file, err := parser.Parse(path, strings.NewReader(string(data)))
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}

	got, want := ast.Fprint(file), string(data)
	if got == want {
		return
	}
	t.Errorf("%s is not canonical; run: tdl fmt -w %s\nfirst difference at %s",
		path, path, firstDiff(got, want))
}

// firstDiff names the line where two renderings part ways, so a failure
// points at a line rather than at two whole files.
func firstDiff(got, want string) string {
	g, w := strings.Split(got, "\n"), strings.Split(want, "\n")
	for i := 0; i < len(g) && i < len(w); i++ {
		if g[i] != w[i] {
			return fmt.Sprintf("line %d:\n  got:  %q\n  want: %q", i+1, g[i], w[i])
		}
	}
	return fmt.Sprintf("end of file: got %d lines, want %d", len(g), len(w))
}

// skipPending skips a corpus case whose directory holds a `pending` file,
// reporting the reason it records.
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
