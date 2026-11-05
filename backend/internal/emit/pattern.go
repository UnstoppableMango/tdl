package emit

import (
	"fmt"
	"regexp/syntax"
	"slices"
	"strings"
	"unicode"
)

// ECMAScript prints a pattern [Pattern] parsed as the body of an
// ECMAScript regex literal with the u flag and no other, matching what
// RE2 matches: case folds and classes are written out as the code points
// Go's Unicode tables give them, `.` is a class, and the line anchors are
// lookaround, so the consumer's engine reads no flag and no Unicode table
// of its own. docs/architecture/zod-backend.md's Patterns is the table.
func ECMAScript(re *syntax.Regexp) string {
	var b strings.Builder
	printECMA(&b, re)
	return b.String()
}

func printECMA(b *strings.Builder, re *syntax.Regexp) {
	switch re.Op {
	case syntax.OpNoMatch:
		b.WriteString("[]")
	case syntax.OpEmptyMatch:
		b.WriteString("(?:)")
	case syntax.OpLiteral:
		for _, r := range re.Rune {
			if re.Flags&syntax.FoldCase != 0 {
				if folds := orbit(r); len(folds) > 1 {
					b.WriteString("[")
					for _, f := range folds {
						b.WriteString(classRune(f))
					}
					b.WriteString("]")
					continue
				}
			}
			b.WriteString(literalRune(r))
		}
	case syntax.OpCharClass:
		printClass(b, re.Rune)
	case syntax.OpAnyCharNotNL:
		b.WriteString(`[^\n]`)
	case syntax.OpAnyChar:
		b.WriteString(`[\u{0}-\u{10FFFF}]`)
	case syntax.OpBeginText:
		b.WriteString("^")
	case syntax.OpEndText:
		b.WriteString("$")
	case syntax.OpBeginLine:
		b.WriteString(`(?<=^|\n)`)
	case syntax.OpEndLine:
		b.WriteString(`(?=\n|$)`)
	case syntax.OpWordBoundary:
		b.WriteString(`\b`)
	case syntax.OpNoWordBoundary:
		b.WriteString(`\B`)
	case syntax.OpCapture:
		// ECMAScript names a group with an identifier, and RE2 also
		// allows one starting with a digit; a group's name never changes
		// what it matches.
		if re.Name != "" && !unicode.IsDigit(rune(re.Name[0])) {
			b.WriteString("(?<" + re.Name + ">")
		} else {
			b.WriteString("(")
		}
		printECMA(b, re.Sub[0])
		b.WriteString(")")
	case syntax.OpStar, syntax.OpPlus, syntax.OpQuest, syntax.OpRepeat:
		printOperand(b, re.Sub[0])
		switch re.Op {
		case syntax.OpStar:
			b.WriteString("*")
		case syntax.OpPlus:
			b.WriteString("+")
		case syntax.OpQuest:
			b.WriteString("?")
		default:
			switch {
			case re.Max == -1:
				fmt.Fprintf(b, "{%d,}", re.Min)
			case re.Min == re.Max:
				fmt.Fprintf(b, "{%d}", re.Min)
			default:
				fmt.Fprintf(b, "{%d,%d}", re.Min, re.Max)
			}
		}
		if re.Flags&syntax.NonGreedy != 0 {
			b.WriteString("?")
		}
	case syntax.OpConcat:
		if len(re.Sub) == 0 {
			b.WriteString("(?:)")
		}
		for _, s := range re.Sub {
			if s.Op == syntax.OpAlternate {
				b.WriteString("(?:")
				printECMA(b, s)
				b.WriteString(")")
			} else {
				printECMA(b, s)
			}
		}
	case syntax.OpAlternate:
		for i, s := range re.Sub {
			if i > 0 {
				b.WriteString("|")
			}
			printECMA(b, s)
		}
	}
}

// printOperand prints a repetition's operand, grouped unless it is one
// atom. ECMAScript under u refuses a quantified assertion, so an anchor
// is grouped too.
func printOperand(b *strings.Builder, re *syntax.Regexp) {
	atom := false
	switch re.Op {
	case syntax.OpCharClass, syntax.OpAnyChar, syntax.OpAnyCharNotNL, syntax.OpCapture, syntax.OpNoMatch, syntax.OpEmptyMatch:
		atom = true
	case syntax.OpLiteral:
		atom = len(re.Rune) == 1
	}
	if atom {
		printECMA(b, re)
		return
	}
	b.WriteString("(?:")
	printECMA(b, re)
	b.WriteString(")")
}

// printClass prints a class's ranges, or the ranges it excludes when they
// are fewer, since under u a negated class is its complement over code
// points.
func printClass(b *strings.Builder, ranges []rune) {
	negated := false
	if len(ranges) > 0 && ranges[0] == 0 && ranges[len(ranges)-1] == unicode.MaxRune {
		if comp := complement(ranges); len(comp) < len(ranges) {
			ranges, negated = comp, true
		}
	}
	b.WriteString("[")
	if negated {
		b.WriteString("^")
	}
	for i := 0; i+1 < len(ranges); i += 2 {
		lo, hi := ranges[i], ranges[i+1]
		b.WriteString(classRune(lo))
		if hi != lo {
			if hi > lo+1 {
				b.WriteString("-")
			}
			b.WriteString(classRune(hi))
		}
	}
	b.WriteString("]")
}

// complement is the code points ranges leaves out.
func complement(ranges []rune) []rune {
	var out []rune
	next := rune(0)
	for i := 0; i+1 < len(ranges); i += 2 {
		if ranges[i] > next {
			out = append(out, next, ranges[i]-1)
		}
		next = ranges[i+1] + 1
	}
	if next <= unicode.MaxRune {
		out = append(out, next, unicode.MaxRune)
	}
	return out
}

// orbit is a rune's simple case folds, itself included, in order.
func orbit(r rune) []rune {
	folds := []rune{r}
	for f := unicode.SimpleFold(r); f != r; f = unicode.SimpleFold(f) {
		folds = append(folds, f)
	}
	slices.Sort(folds)
	return folds
}

// literalRune escapes a rune outside a class: a syntax character or the
// literal's delimiter with a backslash, and anything but printable ASCII
// as a code point escape, which the u flag reads.
func literalRune(r rune) string {
	if strings.ContainsRune(`^$\.*+?()[]{}|/`, r) {
		return `\` + string(r)
	}
	return printable(r)
}

// classRune escapes a rune inside a class.
func classRune(r rune) string {
	if strings.ContainsRune(`\]-[^/`, r) {
		return `\` + string(r)
	}
	return printable(r)
}

// printable writes printable ASCII as itself and every other rune as a
// code point escape, so a pattern reads the same in any editor: a Kelvin
// sign is not mistaken for a K, and no combining mark joins a bracket.
func printable(r rune) string {
	if r >= ' ' && r <= '~' {
		return string(r)
	}
	return fmt.Sprintf(`\u{%X}`, r)
}

// Backtracks reports whether a pattern has a shape a backtracking engine
// can take exponential time on: inside a repetition of more than one, a
// choice
// whose ways can match the same next character, so a failing match tries
// every way of splitting the input. That is a repetition, optional part,
// or alternation branch that can either go on or give way to what follows
// it, as in `(a+)+$`, `(\w+\s?)+$`, and `(a|aa)+$`, which RE2 parses as
// `(a(?:|a))+$`. A bounded repetition counts, since `(a+){2,100}$` is as
// slow as `(a+)+$` on any input a consumer sends. It is a heuristic over
// the tree: polynomial backtracking such as `a*a*a*$` passes.
func Backtracks(re *syntax.Regexp) bool {
	switch re.Op {
	case syntax.OpStar, syntax.OpPlus, syntax.OpRepeat:
		if re.Op != syntax.OpRepeat || re.Max == -1 || re.Max > 1 {
			body := re.Sub[0]
			first, _ := firstOf(body)
			if ambiguous(body, first) {
				return true
			}
		}
	}
	return slices.ContainsFunc(re.Sub, Backtracks)
}

// ambiguous reports whether re, followed by what can start follow, holds a
// choice two of whose ways start with one character.
func ambiguous(re *syntax.Regexp, follow []rune) bool {
	switch re.Op {
	case syntax.OpCapture:
		return ambiguous(re.Sub[0], follow)
	case syntax.OpConcat:
		f := follow
		for i := len(re.Sub) - 1; i >= 0; i-- {
			if ambiguous(re.Sub[i], f) {
				return true
			}
			first, nullable := firstOf(re.Sub[i])
			if nullable {
				f = union(first, f)
			} else {
				f = first
			}
		}
		return false
	case syntax.OpAlternate:
		nullables := 0
		var firsts [][]rune
		for _, s := range re.Sub {
			first, nullable := firstOf(s)
			if nullable {
				nullables++
				// Matching nothing gives way to what follows.
				first = union(first, follow)
			}
			for _, other := range firsts {
				if overlaps(first, other) {
					return true
				}
			}
			firsts = append(firsts, first)
		}
		if nullables > 1 {
			return true
		}
		return slices.ContainsFunc(re.Sub, func(s *syntax.Regexp) bool { return ambiguous(s, follow) })
	case syntax.OpStar, syntax.OpPlus, syntax.OpQuest, syntax.OpRepeat:
		sub := re.Sub[0]
		first, nullable := firstOf(sub)
		// Whether the operand can match a variable number of times.
		varies := re.Op != syntax.OpRepeat || re.Min != re.Max
		if varies && (nullable || overlaps(first, follow)) {
			return true
		}
		if re.Op != syntax.OpQuest && (re.Op != syntax.OpRepeat || re.Max != 1) {
			follow = union(first, follow)
		}
		return ambiguous(sub, follow)
	}
	return false
}

// firstOf is the characters a match of re can start with, and whether it
// can match nothing.
func firstOf(re *syntax.Regexp) ([]rune, bool) {
	switch re.Op {
	case syntax.OpNoMatch:
		return nil, false
	case syntax.OpLiteral:
		if len(re.Rune) == 0 {
			return nil, true
		}
		r := re.Rune[0]
		var set []rune
		if re.Flags&syntax.FoldCase != 0 {
			for _, f := range orbit(r) {
				set = union(set, []rune{f, f})
			}
			return set, false
		}
		return []rune{r, r}, false
	case syntax.OpCharClass:
		return re.Rune, false
	case syntax.OpAnyCharNotNL:
		return []rune{0, '\n' - 1, '\n' + 1, unicode.MaxRune}, false
	case syntax.OpAnyChar:
		return []rune{0, unicode.MaxRune}, false
	case syntax.OpCapture, syntax.OpPlus:
		return firstOf(re.Sub[0])
	case syntax.OpStar, syntax.OpQuest:
		first, _ := firstOf(re.Sub[0])
		return first, true
	case syntax.OpRepeat:
		first, nullable := firstOf(re.Sub[0])
		return first, nullable || re.Min == 0
	case syntax.OpConcat:
		var set []rune
		for _, s := range re.Sub {
			first, nullable := firstOf(s)
			set = union(set, first)
			if !nullable {
				return set, false
			}
		}
		return set, true
	case syntax.OpAlternate:
		var set []rune
		empty := false
		for _, s := range re.Sub {
			first, nullable := firstOf(s)
			set = union(set, first)
			empty = empty || nullable
		}
		return set, empty
	}
	// The empty match, the anchors, and the word boundaries consume
	// nothing.
	return nil, true
}

// union merges two sorted range lists.
func union(a, b []rune) []rune {
	if len(a) == 0 {
		return b
	}
	if len(b) == 0 {
		return a
	}
	type span struct{ lo, hi rune }
	var spans []span
	for _, rs := range [][]rune{a, b} {
		for i := 0; i+1 < len(rs); i += 2 {
			spans = append(spans, span{rs[i], rs[i+1]})
		}
	}
	slices.SortFunc(spans, func(x, y span) int { return int(x.lo - y.lo) })
	var out []rune
	for _, s := range spans {
		if n := len(out); n > 0 && s.lo <= out[n-1]+1 {
			out[n-1] = max(out[n-1], s.hi)
			continue
		}
		out = append(out, s.lo, s.hi)
	}
	return out
}

// overlaps reports whether two range lists share a character.
func overlaps(a, b []rune) bool {
	for i := 0; i+1 < len(a); i += 2 {
		for j := 0; j+1 < len(b); j += 2 {
			if a[i] <= b[j+1] && b[j] <= a[i+1] {
				return true
			}
		}
	}
	return false
}
