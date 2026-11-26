package plugin

import (
	"path"
	"strings"
)

// MatchImport reports whether an importer declaring patterns reads the
// file with base name name. Each pattern is a [path.Match] glob; one
// starting with `!` excludes what it matches. A name is read when an
// include matches it and no exclude does, so the order of patterns does
// not matter. A malformed pattern matches nothing.
func MatchImport(patterns []string, name string) bool {
	included := false
	for _, p := range patterns {
		glob, exclude := strings.CutPrefix(p, "!")
		ok, err := path.Match(glob, name)
		if err != nil || !ok {
			continue
		}
		if exclude {
			return false
		}
		included = true
	}
	return included
}
