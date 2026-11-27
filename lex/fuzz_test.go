package lex_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/unstoppablemango/tdl/lex"
)

// FuzzLex checks that lexing arbitrary input never panics and reaches EOF,
// each token ending past the one before it. The corpora seed it, so the
// seeds run in every go test.
func FuzzLex(f *testing.F) {
	seedCorpora(f)
	f.Fuzz(func(t *testing.T, src string) {
		l := lex.New("fuzz.tdl", src)
		prev := -1
		for range len(src) + 1 {
			tok := l.Next()
			if tok.Kind == lex.EOF {
				return
			}
			if tok.End.Offset <= prev || tok.End.Offset > len(src) {
				t.Fatalf("%v token ends at %d after a token ending at %d, in %d bytes", tok.Kind, tok.End.Offset, prev, len(src))
			}
			prev = tok.End.Offset
		}
		t.Fatalf("no EOF after %d tokens", len(src)+1)
	})
}

// seedCorpora adds every source.tdl in the conformance and invalid corpora.
func seedCorpora(f *testing.F) {
	f.Helper()
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
}
