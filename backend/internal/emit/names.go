package emit

import (
	"strings"
	"unicode"
)

// LastSegment is the part of a dotted name after its last dot. A
// declaration's name may arrive qualified, and every target writes the bare
// name.
func LastSegment(name string) string {
	if i := strings.LastIndex(name, "."); i >= 0 {
		return name[i+1:]
	}
	return name
}

// Words splits a name into the words a case convention joins.
//
// A boundary falls before an upper-case letter that follows a lower-case
// letter or a digit, and before the last letter of an upper-case run that a
// lower-case letter follows, so `userID` is user and ID and `HTTPServer` is
// HTTP and Server. Anything that is not a letter or a digit separates words
// and is dropped. A digit stays with the word it follows: `last4` is one
// word.
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

// Pascal joins words with each one's first letter upper case and the rest
// as written: `user_id` is UserId and `userID` is UserID.
func Pascal(name string) string {
	var b strings.Builder
	for _, w := range Words(name) {
		b.WriteString(upperFirst(w))
	}
	return b.String()
}

// Camel is [Pascal] with the first word entirely lower case.
func Camel(name string) string {
	words := Words(name)
	if len(words) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString(strings.ToLower(words[0]))
	for _, w := range words[1:] {
		b.WriteString(upperFirst(w))
	}
	return b.String()
}

// Snake joins lower-case words with underscores.
func Snake(name string) string {
	words := Words(name)
	for i, w := range words {
		words[i] = strings.ToLower(w)
	}
	return strings.Join(words, "_")
}

// ScreamingSnake joins upper-case words with underscores.
func ScreamingSnake(name string) string {
	words := Words(name)
	for i, w := range words {
		words[i] = strings.ToUpper(w)
	}
	return strings.Join(words, "_")
}

func upperFirst(w string) string {
	r := []rune(w)
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}

func trim(s string) string { return strings.TrimSpace(s) }
