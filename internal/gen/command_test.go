package gen

import (
	"slices"
	"testing"
)

// A command line splits into words as a shell would, with nothing
// expanded.
func TestSplitWords(t *testing.T) {
	for _, tt := range []struct {
		line string
		want []string
	}{
		{"tr a-z A-Z", []string{"tr", "a-z", "A-Z"}},
		{"  biome\tformat  --stdin-file-path={file} ", []string{"biome", "format", "--stdin-file-path={file}"}},
		{`sh -c 'echo "$0"; exit 1' {file}`, []string{"sh", "-c", `echo "$0"; exit 1`, "{file}"}},
		{`echo "a \"b\" \$c \d"`, []string{"echo", `a "b" $c \d`}},
		{`a\ b c`, []string{"a b", "c"}},
		{`x''y ""`, []string{"xy", ""}},
	} {
		got, err := splitWords(tt.line)
		if err != nil {
			t.Errorf("%s: %v", tt.line, err)
			continue
		}
		if !slices.Equal(got, tt.want) {
			t.Errorf("%s: words = %q, want %q", tt.line, got, tt.want)
		}
	}
}

func TestSplitWordsRefuses(t *testing.T) {
	for _, line := range []string{"", "   ", "'open", `"open`, `trailing\`} {
		if words, err := splitWords(line); err == nil {
			t.Errorf("%q split into %q", line, words)
		}
	}
}
