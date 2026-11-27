package parser_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/unstoppablemango/tdl/ast"
	"github.com/unstoppablemango/tdl/parser"
)

// FuzzParse checks that parsing arbitrary input never panics, and that
// input parsing without error prints to source that parses again and
// prints the same bytes a second time, so tdl fmt is idempotent. The
// corpora seed it, so the seeds run in every go test.
func FuzzParse(f *testing.F) {
	for _, corpus := range []string{"conformance", "invalid"} {
		paths, err := filepath.Glob(filepath.Join("..", "testdata", corpus, "*", "source.tdl"))
		if err != nil {
			f.Fatal(err)
		}
		if len(paths) == 0 {
			f.Fatalf("no seeds in testdata/%s", corpus)
		}
		for _, path := range paths {
			data, err := os.ReadFile(path)
			if err != nil {
				f.Fatal(err)
			}
			f.Add(string(data))
		}
	}
	f.Fuzz(func(t *testing.T, src string) {
		file, err := parser.Parse("fuzz.tdl", strings.NewReader(src))
		if err != nil {
			return
		}
		once := ast.Fprint(file)
		again, err := parser.Parse("fuzz.tdl", strings.NewReader(once))
		if err != nil {
			t.Fatalf("printed source does not parse: %v\n%s", err, once)
		}
		if twice := ast.Fprint(again); twice != once {
			t.Fatalf("printing is not idempotent\nfirst:\n%s\nsecond:\n%s", once, twice)
		}
	})
}
