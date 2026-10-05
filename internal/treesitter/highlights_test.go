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

// TestHighlightsCoverKeywords checks that queries/highlights.scm spells
// every lex keyword. A keyword is an anonymous token, so compiling the query
// cannot catch a missing one.
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
