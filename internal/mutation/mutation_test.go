package mutation

import (
	"bytes"
	"reflect"
	"strings"
	"testing"
)

const lexReport = `{
  "go_module": "github.com/unstoppablemango/tdl",
  "files": [{
    "file_name": "lexer.go",
    "mutations": [
      {"type": "CONDITIONALS_BOUNDARY", "status": "LIVED", "line": 82, "column": 38},
      {"type": "INVERT_LOGICAL", "status": "KILLED", "line": 82, "column": 19},
      {"type": "INVERT_NEGATIVES", "status": "NOT COVERED", "line": 47, "column": 13},
      {"type": "INVERT_LOOPCTRL", "status": "LIVED", "line": 12, "column": 3},
      {"type": "REMOVE_SELF_ASSIGNMENTS", "status": "TIMED OUT", "line": 61, "column": 13},
      {"type": "ARITHMETIC_BASE", "status": "NOT VIABLE", "line": 9, "column": 2}
    ]
  }],
  "elapsed_time": 150.5
}`

const parserReport = `{
  "go_module": "github.com/unstoppablemango/tdl",
  "files": [{
    "file_name": "parser.go",
    "mutations": [
      {"type": "CONDITIONALS_NEGATION", "status": "LIVED", "line": 5, "column": 7}
    ]
  }],
  "elapsed_time": 10
}`

func merged(t *testing.T) *Report {
	t.Helper()
	lex, err := Read(strings.NewReader(lexReport))
	if err != nil {
		t.Fatal(err)
	}
	parser, err := Read(strings.NewReader(parserReport))
	if err != nil {
		t.Fatal(err)
	}
	return Merge([]string{"lex", "parser"}, []*Report{lex, parser})
}

func TestMergeNamesFilesFromTheModuleRoot(t *testing.T) {
	rep := merged(t)
	var names []string
	for _, f := range rep.Files {
		names = append(names, f.FileName)
	}
	if want, got := []string{"lex/lexer.go", "parser/parser.go"}, names; !reflect.DeepEqual(want, got) {
		t.Errorf("file names got\n%#v\nwant\n%#v", got, want)
	}
	if rep.ElapsedTime != 160.5 {
		t.Errorf("elapsed time = %v, want 160.5", rep.ElapsedTime)
	}
}

func TestWriteReadsBack(t *testing.T) {
	rep := merged(t)
	var b bytes.Buffer
	if err := Write(&b, rep); err != nil {
		t.Fatal(err)
	}
	got, err := Read(&b)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(rep, got) {
		t.Errorf("round trip got\n%#v\nwant\n%#v", got, rep)
	}
}

func TestMutatorMatchesTheConfiguration(t *testing.T) {
	for typ, want := range map[string]string{
		"CONDITIONALS_BOUNDARY":   "conditionals-boundary",
		"INVERT_LOOPCTRL":         "invert-loopctrl",
		"REMOVE_SELF_ASSIGNMENTS": "remove-self-assignments",
	} {
		if got := Mutator(typ); got != want {
			t.Errorf("Mutator(%q) = %q, want %q", typ, got, want)
		}
	}
}

func TestParseEquivalent(t *testing.T) {
	got, err := ParseEquivalent(strings.NewReader(`# A comment.

# Why this one cannot be killed.
lex/lexer.go:82:38 conditionals-boundary
  parser/parser.go:5:7 conditionals-negation
`))
	if err != nil {
		t.Fatal(err)
	}
	want := []Mutant{
		{File: "lex/lexer.go", Line: 82, Column: 38, Mutator: "conditionals-boundary"},
		{File: "parser/parser.go", Line: 5, Column: 7, Mutator: "conditionals-negation"},
	}
	if !reflect.DeepEqual(want, got) {
		t.Errorf("got\n%#v\nwant\n%#v", got, want)
	}
}

func TestParseEquivalentRefusesMalformedLines(t *testing.T) {
	for _, line := range []string{
		"lex/lexer.go:82:38",
		"lex/lexer.go:82 conditionals-boundary",
		"lex/lexer.go:x:38 conditionals-boundary",
		"lex/lexer.go:82:0 conditionals-boundary",
		":82:38 conditionals-boundary",
		"lex/lexer.go:82:38 conditionals-boundary extra",
	} {
		_, err := ParseEquivalent(strings.NewReader("# ok\n" + line + "\n"))
		if err == nil {
			t.Errorf("%q: no error", line)
			continue
		}
		if !strings.HasPrefix(err.Error(), "line 2: ") {
			t.Errorf("%q: error %q does not name line 2", line, err)
		}
	}
}

func TestSummarize(t *testing.T) {
	eq := []Mutant{
		{File: "lex/lexer.go", Line: 82, Column: 38, Mutator: "conditionals-boundary"},
		// Moved or removed: the report holds nothing here.
		{File: "lex/lexer.go", Line: 100, Column: 1, Mutator: "invert-logical"},
		// In a package the run did not cover, so neither applied nor stale.
		{File: "internal/sema/lower.go", Line: 1, Column: 1, Mutator: "invert-logical"},
	}
	got := Summarize(merged(t), []string{"lex", "parser"}, eq)
	want := Summary{
		Survivors: []Mutant{
			{File: "lex/lexer.go", Line: 12, Column: 3, Mutator: "invert-loopctrl"},
			{File: "parser/parser.go", Line: 5, Column: 7, Mutator: "conditionals-negation"},
		},
		NotCovered: []Mutant{
			{File: "lex/lexer.go", Line: 47, Column: 13, Mutator: "invert-negatives"},
		},
		Stale: []Mutant{
			{File: "lex/lexer.go", Line: 100, Column: 1, Mutator: "invert-logical"},
		},
		Counts: []Counts{
			{Package: "lex", Killed: 1, Lived: 1, Equivalent: 1, TimedOut: 1, NotCovered: 1, NotViable: 1},
			{Package: "parser", Lived: 1},
		},
	}
	if !reflect.DeepEqual(want, got) {
		t.Errorf("got\n%#v\nwant\n%#v", got, want)
	}
}

func TestSummarizeCountsAPackageWithNoMutants(t *testing.T) {
	got := Summarize(&Report{}, []string{"ast"}, nil)
	if want, got := []Counts{{Package: "ast"}}, got.Counts; !reflect.DeepEqual(want, got) {
		t.Errorf("got\n%#v\nwant\n%#v", got, want)
	}
}

func TestPrint(t *testing.T) {
	eq := []Mutant{
		{File: "lex/lexer.go", Line: 82, Column: 38, Mutator: "conditionals-boundary"},
		{File: "lex/lexer.go", Line: 100, Column: 1, Mutator: "invert-logical"},
	}
	var b strings.Builder
	if err := Summarize(merged(t), []string{"lex", "parser"}, eq).Print(&b); err != nil {
		t.Fatal(err)
	}
	want := `Survivors:
lex/lexer.go:12:3 invert-loopctrl
parser/parser.go:5:7 conditionals-negation

Not covered:
lex/lexer.go:47:13 invert-negatives

Stale in mutation/equivalent.txt:
lex/lexer.go:100:1 invert-logical

package  killed  lived  equivalent  timed out  not covered  not viable
lex      1       1      1           1          1            1
parser   0       1      0           0          0            0
`
	if got := b.String(); got != want {
		t.Errorf("got\n%s\nwant\n%s", got, want)
	}
}
