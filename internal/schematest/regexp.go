// Package schematest holds what tests compiling a generated JSON Schema or
// OpenAPI document share.
package schematest

import (
	"fmt"
	"regexp"
	"strings"

	sjs "github.com/santhosh-tekuri/jsonschema/v6"
)

// Engine compiles a `pattern` for santhosh-tekuri/jsonschema, which
// compiles one with Go's regexp unless told otherwise. A generated pattern
// is ECMAScript, printed by emit.ECMAScriptSource, which RE2 cannot read,
// so Engine reads back the few spellings that printer writes and RE2 does
// not: `\uXXXX`, `\u{...}`, and the lookaround standing for the line
// anchors. Anything else ECMAScript-only fails to compile, so a pattern
// copied as written in RE2 syntax still compiles and one the printer
// mangled does not. What a printed pattern means to an ECMAScript engine
// is held to RE2 by emit's tests under node; this only lets the
// documents' instance tests run.
func Engine(s string) (sjs.Regexp, error) {
	var b strings.Builder
	for i := 0; i < len(s); {
		switch {
		case strings.HasPrefix(s[i:], `(?<=^|\n)`):
			b.WriteString(`(?m:^)`)
			i += len(`(?<=^|\n)`)
		case strings.HasPrefix(s[i:], `(?=\n|$)`):
			b.WriteString(`(?m:$)`)
			i += len(`(?=\n|$)`)
		case strings.HasPrefix(s[i:], `\u{`):
			end := strings.IndexByte(s[i:], '}')
			if end < 0 {
				return nil, fmt.Errorf("unterminated \\u{ in %s", s)
			}
			b.WriteString(`\x{` + s[i+3:i+end] + `}`)
			i += end + 1
		case strings.HasPrefix(s[i:], `\u`):
			if len(s) < i+6 {
				return nil, fmt.Errorf("short \\u escape in %s", s)
			}
			b.WriteString(`\x{` + s[i+2:i+6] + `}`)
			i += 6
		case s[i] == '\\' && i+1 < len(s):
			b.WriteString(s[i : i+2])
			i += 2
		default:
			b.WriteByte(s[i])
			i++
		}
	}
	return regexp.Compile(b.String())
}
