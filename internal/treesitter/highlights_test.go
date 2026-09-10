package treesitter_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/unstoppablemango/tdl/lex"
)

var highlightsPath = filepath.Join("..", "..", "tree-sitter", "queries", "highlights.scm")

// TestHighlightsCoverKeywords holds the hand-written highlight query to the
// lexer, the way internal/ebnf holds the grammar's quoted terminals to it.
//
// A keyword is an anonymous token, so nothing in the tree carries its name
// and `tree-sitter query` cannot tell a missing one from a deliberate
// omission. A keyword added to lex and not to the query is therefore
// invisible until someone opens a file and notices the color.
//
// The query is read as text rather than parsed, since this asserts that a
// spelling is present and not where it sits.
func TestHighlightsCoverKeywords(t *testing.T) {
	src, err := os.ReadFile(highlightsPath)
	if err != nil {
		t.Fatalf("reading %s: %v", highlightsPath, err)
	}
	query := string(src)

	for _, kw := range lex.Keywords() {
		if !strings.Contains(query, fmt.Sprintf("%q", kw)) {
			t.Errorf("%s does not capture the keyword %q", highlightsPath, kw)
		}
	}
}
