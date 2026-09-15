package irtest

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/unstoppablemango/tdl/internal/sema"
	"github.com/unstoppablemango/tdl/ir"
	"github.com/unstoppablemango/tdl/parser"
)

// Conformance parses and lowers the source.tdl of a conformance case
// directory, reading its imports from beside it. The corpus lowers with no
// diagnostic, so any diagnostic fails the test.
func Conformance(t testing.TB, dir string) *ir.Model {
	t.Helper()

	path := filepath.Join(dir, "source.tdl")
	src, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	file, err := parser.Parse(path, bytes.NewReader(src))
	if err != nil {
		t.Fatalf("parsing %s: %v", path, err)
	}
	model, diags := sema.Lower(file, sema.WithLoader(sema.FSLoader{}))
	if len(diags) > 0 {
		t.Fatalf("lowering %s: %v", path, diags.Error())
	}
	return model
}
