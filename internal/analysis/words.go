package analysis

import (
	"strings"
	"unicode"
)

// Words splits a name into the words a case convention joins: `userID` is
// user and ID, `HTTPServer` is HTTP and Server, and `last4` is one word.
// Anything that is not a letter or a digit separates words and is dropped.
func Words(name string) []string {
	var words []string
	var cur []rune
	flush := func() {
		if len(cur) > 0 {
			words = append(words, string(cur))
			cur = nil
		}
	}

	r := []rune(name)
	for i, c := range r {
		if !unicode.IsLetter(c) && !unicode.IsDigit(c) {
			flush()
			continue
		}
		if unicode.IsUpper(c) && len(cur) > 0 {
			prev := cur[len(cur)-1]
			nextLower := i+1 < len(r) && unicode.IsLower(r[i+1])
			if unicode.IsLower(prev) || unicode.IsDigit(prev) || (unicode.IsUpper(prev) && nextLower) {
				flush()
			}
		}
		cur = append(cur, c)
	}
	flush()
	return words
}

func lastSegment(name string) string {
	if i := strings.LastIndex(name, "."); i >= 0 {
		return name[i+1:]
	}
	return name
}
