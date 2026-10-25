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

// Call writes each argument as a target block spells it, so a name argument,
// which never resolves to a variant, carries no dump annotation.
func TestCallWritesArgumentsAsSource(t *testing.T) {
	low := int64(1)
	d := &Directive{
		Name: "foo",
		Args: []*Literal{
			{Kind: LiteralKind_LITERAL_KIND_NAME, Text: "bar"},
			{Kind: LiteralKind_LITERAL_KIND_STRING, Text: "s"},
			{Kind: LiteralKind_LITERAL_KIND_LIST, Items: []*Literal{
				{Kind: LiteralKind_LITERAL_KIND_NAME, Text: "baz"},
				{Kind: LiteralKind_LITERAL_KIND_INT, Text: "2"},
			}},
			{Kind: LiteralKind_LITERAL_KIND_RANGE, Range: &Range{Low: &low}},
			{Kind: LiteralKind_LITERAL_KIND_REGEX, Text: "^a$"},
		},
	}

	want := `foo(bar, "s", [baz, 2], 1.., /^a$/)`
	if got := d.Call(); got != want {
		t.Errorf("Call() = %q, want %q", got, want)
	}
}
