package ir

import (
	"strings"
	"testing"
)

// Every kind, including the zero value and an unknown one, carries an
// article.
func TestKindNameIsASentenceFragment(t *testing.T) {
	kinds := []LiteralKind{
		LiteralKind_LITERAL_KIND_UNSPECIFIED,
		LiteralKind_LITERAL_KIND_STRING,
		LiteralKind_LITERAL_KIND_INT,
		LiteralKind_LITERAL_KIND_FLOAT,
		LiteralKind_LITERAL_KIND_BOOL,
		LiteralKind_LITERAL_KIND_NAME,
		LiteralKind_LITERAL_KIND_REGEX,
		LiteralKind_LITERAL_KIND_LIST,
		LiteralKind_LITERAL_KIND_RANGE,
		LiteralKind(99), // a kind from a schema newer than this build
	}

	for _, k := range kinds {
		got := KindName(k)
		if !strings.HasPrefix(got, "a ") && !strings.HasPrefix(got, "an ") {
			t.Errorf("KindName(%v) = %q, which does not read into a sentence", k, got)
		}
	}
}
