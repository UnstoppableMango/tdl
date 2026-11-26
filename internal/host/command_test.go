package host

import (
	"slices"
	"testing"
)

func TestSplitWords(t *testing.T) {
	tests := []struct {
		line string
		want []string
	}{
		{"tr a-z A-Z", []string{"tr", "a-z", "A-Z"}},
		{"  biome\tformat  --stdin-file-path={file} ", []string{"biome", "format", "--stdin-file-path={file}"}},
		{`sed 's/a b/c d/'`, []string{"sed", "s/a b/c d/"}},
		{`printf "%s \"x\"" ''`, []string{"printf", `%s "x"`, ""}},
		{`a\ b c`, []string{"a b", "c"}},
		{`x'y'"z"`, []string{"xyz"}},
		{`echo "C:\tmp \" \\" 'a\b'`, []string{"echo", `C:\tmp " \`, `a\b`}},
	}
	for _, tc := range tests {
		got, err := splitWords(tc.line)
		if err != nil || !slices.Equal(got, tc.want) {
			t.Errorf("splitWords(%q) = %q, %v; want %q", tc.line, got, err, tc.want)
		}
	}

	for _, bad := range []string{"", "   ", "tr 'a", `tr "a`, `tr \`} {
		if got, err := splitWords(bad); err == nil {
			t.Errorf("splitWords(%q) = %q, want an error", bad, got)
		}
	}
}
