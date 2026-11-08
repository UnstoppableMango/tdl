package emit_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"regexp/syntax"
	"strings"
	"testing"

	"github.com/unstoppablemango/tdl/backend/internal/emit"
)

// A patternCase is a pattern in RE2 syntax and inputs to match it
// against. testdata/patterns.json is shared by every engine's printer:
// each must agree with Go's regexp on every input.
type patternCase struct {
	Pattern string   `json:"pattern"`
	Inputs  []string `json:"inputs"`
}

func patternCases(t *testing.T) []patternCase {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "patterns.json"))
	if err != nil {
		t.Fatal(err)
	}
	var cases []patternCase
	if err := json.Unmarshal(data, &cases); err != nil {
		t.Fatal(err)
	}
	return cases
}

func parsePattern(t *testing.T, p string) *syntax.Regexp {
	t.Helper()
	re, err := syntax.Parse(p, syntax.Perl)
	if err != nil {
		t.Fatalf("%s: %v", p, err)
	}
	return re
}

func TestECMAScriptSpelling(t *testing.T) {
	for _, c := range []struct{ pattern, want string }{
		{`\s`, `[\u{9}\u{A}\u{C}\u{D} ]`},
		{`(?i)k`, `[Kk\u{212A}]`},
		{`\d`, `[0-9]`},
		{`.`, `[^\n]`},
		{`(?s).`, `[\u{0}-\u{10FFFF}]`},
		{`[^@]`, `[^@]`},
		{`(?m)^a$`, `(?<=^|\n)a(?=\n|$)`},
		{`^a/b$`, `^a\/b$`},
		{`(?P<year>\d{4})`, `(?<year>[0-9]{4})`},
		{`(?P<a>x)(?P<a>y)`, `(?<a>x)(y)`},
		{`(ab)+?`, `(ab)+?`},
		{`(?:ab)*`, `(?:ab)*`},
		{`a|b+c`, `a|b+c`},
		{`x(?:a|b+)`, `x(?:a|b+)`},
		{`a{2,}`, `a{2,}`},
	} {
		if got := emit.ECMAScript(parsePattern(t, c.pattern)); got != c.want {
			t.Errorf("%s: %s, want %s", c.pattern, got, c.want)
		}
	}
}

func TestECMAScriptSourceSpelling(t *testing.T) {
	for _, c := range []struct{ pattern, want string }{
		{`\s`, `[\u0009\u000A\u000C\u000D ]`},
		{`(?i)k`, `[Kk\u212A]`},
		{`(?s).`, `[\u0000-\u{10FFFF}]`},
		{`^a/b$`, `^a/b$`},
		{`[/a]`, `[/a]`},
		{`\x{1F600}`, `\u{1F600}`},
		{`(?m)^a$`, `(?<=^|\n)a(?=\n|$)`},
	} {
		if got := emit.ECMAScriptSource(parsePattern(t, c.pattern)); got != c.want {
			t.Errorf("%s: %s, want %s", c.pattern, got, c.want)
		}
	}
}

func TestNeedsUnicode(t *testing.T) {
	for _, c := range []struct {
		pattern string
		want    bool
	}{
		{`^[a-z]+$`, false},
		{`\s`, false},
		{`(?i)k`, false},
		// Greek reaches U+10140, Ancient Greek Numbers.
		{`\p{Greek}`, true},
		{`[\x{D000}-\x{E000}]`, true},
		{`(?m)^a$`, false},
		{`^.$`, true},
		{`(?s).`, true},
		{`[^a]`, true},
		{`\S`, true},
		{`\x{1F600}`, true},
		{`[a\x{1F600}]`, true},
		{`[\x{E000}-\x{FFFF}]`, false},
		{`\pL`, true},
	} {
		if got := emit.NeedsUnicode(parsePattern(t, c.pattern)); got != c.want {
			t.Errorf("NeedsUnicode(%s) = %v, want %v", c.pattern, got, c.want)
		}
	}
}

func TestSurrogates(t *testing.T) {
	for _, c := range []struct {
		pattern string
		want    bool
	}{
		{`\x{D800}`, true},
		{`[\x{DC00}-\x{DFFF}]`, true},
		{`[a\x{D7FF}-\x{D800}]`, true},
		{`(?s).`, false},
		{`[^a]`, false},
		{`\x{E000}`, false},
	} {
		if got := emit.Surrogates(parsePattern(t, c.pattern)); got != c.want {
			t.Errorf("%s: %v, want %v", c.pattern, got, c.want)
		}
	}
}

func TestBacktracks(t *testing.T) {
	for _, c := range []struct {
		pattern string
		want    bool
	}{
		{`(a+)+$`, true},
		{`(a|aa)+$`, true},
		{`(a*)*b`, true},
		{`(\w+\s?)+$`, true},
		{`(a|a?)+`, true},
		{`(?:a{1,3})+$`, true},
		{`(a+){2,100}$`, true},
		{`(a|aa){1,50}$`, true},
		{`(?:a?){0,64}a{0,64}$`, true},
		{`^(\d{3}-){2}\d{4}$`, false},
		{`^[a-z]{1,10}$`, false},
		{`^[^@]+@[^@]+$`, false},
		// RE2 merges the branches into one class.
		{`(?i)(?:k|\x{212A})+$`, false},
		{`^([a-z0-9]+-)*[a-z0-9]+$`, false},
		{`^(?:ab|cd)+$`, false},
		{`^(ab?)+$`, false},
		{`^(\d{3}-)*\d{4}$`, false},
		// Polynomial, which the heuristic lets through.
		{`a*a*a*$`, false},
		{`^[A-Z0-9-]+$`, false},
	} {
		if got := emit.Backtracks(parsePattern(t, c.pattern)); got != c.want {
			t.Errorf("Backtracks(%s) = %v, want %v", c.pattern, got, c.want)
		}
	}
}

// TestECMAScriptAgreesWithRE2 runs every printed pattern under node and
// compares what it matches with Go's regexp, unanchored as matches reads:
// [emit.ECMAScript] as a regex literal with the u flag, and
// [emit.ECMAScriptSource] as a RegExp's source with the flag and, where
// [emit.NeedsUnicode] says the flag changes nothing, without it.
func TestECMAScriptAgreesWithRE2(t *testing.T) {
	cases := patternCases(t)
	if testing.Short() {
		t.Skip("-short")
	}
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node is not on PATH")
	}

	type form struct {
		name    string
		printed string
		// js is the expression building the regex.
		js string
	}
	forms := make([][]form, len(cases))
	var script strings.Builder
	script.WriteString("const cases = [\n")
	for i, c := range cases {
		re := parsePattern(t, c.Pattern)
		lit, src := emit.ECMAScript(re), emit.ECMAScriptSource(re)
		quoted, err := json.Marshal(src)
		if err != nil {
			t.Fatal(err)
		}
		forms[i] = []form{
			{"literal", "/" + lit + "/u", "/" + lit + "/u"},
			{"source with u", src, "new RegExp(" + string(quoted) + ", \"u\")"},
		}
		if !emit.NeedsUnicode(re) {
			forms[i] = append(forms[i], form{"source without u", src, "new RegExp(" + string(quoted) + ")"})
		}
		inputs, err := json.Marshal(c.Inputs)
		if err != nil {
			t.Fatal(err)
		}
		script.WriteString("  [[")
		for j, f := range forms[i] {
			if j > 0 {
				script.WriteString(", ")
			}
			script.WriteString(f.js)
		}
		script.WriteString("], " + string(inputs) + "],\n")
	}
	script.WriteString("];\nconsole.log(JSON.stringify(cases.map(([res, inputs]) => res.map((re) => inputs.map((s) => re.test(s))))));\n")
	path := filepath.Join(t.TempDir(), "patterns.mjs")
	if err := os.WriteFile(path, []byte(script.String()), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := exec.Command(node, path).Output()
	if err != nil {
		t.Fatalf("node: %v\n%s", err, script.String())
	}
	var got [][][]bool
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("%v: %s", err, out)
	}
	for i, c := range cases {
		re := regexp.MustCompile(c.Pattern)
		for k, f := range forms[i] {
			for j, in := range c.Inputs {
				if want := re.MatchString(in); got[i][k][j] != want {
					t.Errorf("%s %s on %q: node %v, RE2 %v (from %s)", f.name, f.printed, in, got[i][k][j], want, c.Pattern)
				}
			}
		}
	}
}
