// Package mutation reads gremlins' JSON reports and the list of
// equivalent mutants, and prints what a run left alive.
//
// See docs/architecture/mutation-testing.md.
package mutation

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"path"
	"slices"
	"strconv"
	"strings"
	"text/tabwriter"
)

// The statuses gremlins writes to a report.
const (
	Killed     = "KILLED"
	Lived      = "LIVED"
	TimedOut   = "TIMED OUT"
	NotCovered = "NOT COVERED"
	NotViable  = "NOT VIABLE"
	Skipped    = "SKIPPED"
)

// Report is gremlins' JSON report, holding only the fields read here.
// A report from one package names its files relative to that package;
// Merge rewrites them relative to the module root.
type Report struct {
	GoModule    string  `json:"go_module"`
	Files       []File  `json:"files"`
	ElapsedTime float64 `json:"elapsed_time"`
}

// File is one source file's mutants.
type File struct {
	FileName  string     `json:"file_name"`
	Mutations []Mutation `json:"mutations"`
}

// Mutation is one mutant.
// Type is gremlins' constant, such as CONDITIONALS_BOUNDARY.
type Mutation struct {
	Type   string `json:"type"`
	Status string `json:"status"`
	Line   int    `json:"line"`
	Column int    `json:"column"`
}

// Read decodes one report.
func Read(r io.Reader) (*Report, error) {
	var rep Report
	if err := json.NewDecoder(r).Decode(&rep); err != nil {
		return nil, err
	}
	return &rep, nil
}

// Write encodes a report, indented so a diff of two runs reads.
func Write(w io.Writer, rep *Report) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(rep)
}

// Merge combines per-package reports into one whose file names are
// relative to the module root: pkgs[i] is the directory reps[i] ran in.
func Merge(pkgs []string, reps []*Report) *Report {
	out := &Report{}
	for i, rep := range reps {
		if out.GoModule == "" {
			out.GoModule = rep.GoModule
		}
		out.ElapsedTime += rep.ElapsedTime
		for _, f := range rep.Files {
			f.FileName = path.Join(pkgs[i], f.FileName)
			out.Files = append(out.Files, f)
		}
	}
	return out
}

// Mutant identifies a mutant the way the report does: a file relative
// to the module root, a position, and a mutator.
type Mutant struct {
	File    string
	Line    int
	Column  int
	Mutator string
}

func (m Mutant) String() string {
	return fmt.Sprintf("%s:%d:%d %s", m.File, m.Line, m.Column, m.Mutator)
}

// Mutator spells a report's mutator constant as the configuration does:
// CONDITIONALS_BOUNDARY is conditionals-boundary.
func Mutator(typ string) string {
	return strings.ReplaceAll(strings.ToLower(typ), "_", "-")
}

// ParseEquivalent reads mutation/equivalent.txt: one mutant per line as
// file:line:column mutator, with blank lines and # comments ignored.
func ParseEquivalent(r io.Reader) ([]Mutant, error) {
	var out []Mutant
	sc := bufio.NewScanner(r)
	for n := 1; sc.Scan(); n++ {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		m, err := parseMutant(line)
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", n, err)
		}
		out = append(out, m)
	}
	return out, sc.Err()
}

func parseMutant(s string) (Mutant, error) {
	pos, mutator, ok := strings.Cut(s, " ")
	mutator = strings.TrimSpace(mutator)
	if !ok || mutator == "" || strings.ContainsAny(mutator, " \t") {
		return Mutant{}, fmt.Errorf("%q: want file:line:column mutator", s)
	}
	parts := strings.Split(pos, ":")
	if len(parts) != 3 || parts[0] == "" {
		return Mutant{}, fmt.Errorf("%q: want file:line:column mutator", s)
	}
	line, err := strconv.Atoi(parts[1])
	if err != nil || line < 1 {
		return Mutant{}, fmt.Errorf("%q: bad line %q", s, parts[1])
	}
	col, err := strconv.Atoi(parts[2])
	if err != nil || col < 1 {
		return Mutant{}, fmt.Errorf("%q: bad column %q", s, parts[2])
	}
	return Mutant{File: parts[0], Line: line, Column: col, Mutator: mutator}, nil
}

// Counts are one package's mutants by status.
// Lived leaves out the equivalent mutants, which Equivalent counts.
type Counts struct {
	Package    string
	Killed     int
	Lived      int
	Equivalent int
	TimedOut   int
	NotCovered int
	NotViable  int
}

// Summary is what a run left: the survivors not listed as equivalent,
// the mutants no test reaches, the listed mutants the report no longer
// holds, and the counts per package.
type Summary struct {
	Survivors  []Mutant
	NotCovered []Mutant
	Stale      []Mutant
	Counts     []Counts
}

// Summarize reads a merged report against the equivalent list.
// pkgs are the directories the run covered: a listed mutant in another
// package is neither applied nor stale, since the run did not look.
func Summarize(rep *Report, pkgs []string, equivalent []Mutant) Summary {
	listed := make(map[Mutant]bool, len(equivalent))
	for _, m := range equivalent {
		listed[m] = false
	}
	counts := map[string]*Counts{}
	for _, p := range pkgs {
		counts[p] = &Counts{Package: p}
	}

	var s Summary
	for _, f := range rep.Files {
		pkg := path.Dir(f.FileName)
		c := counts[pkg]
		if c == nil {
			c = &Counts{Package: pkg}
			counts[pkg] = c
		}
		for _, mu := range f.Mutations {
			m := Mutant{File: f.FileName, Line: mu.Line, Column: mu.Column, Mutator: Mutator(mu.Type)}
			if _, ok := listed[m]; ok {
				listed[m] = true
			}
			switch mu.Status {
			case Killed:
				c.Killed++
			case Lived:
				if _, ok := listed[m]; ok {
					c.Equivalent++
					continue
				}
				c.Lived++
				s.Survivors = append(s.Survivors, m)
			case TimedOut:
				c.TimedOut++
			case NotCovered:
				c.NotCovered++
				s.NotCovered = append(s.NotCovered, m)
			case NotViable:
				c.NotViable++
			}
		}
	}

	for _, m := range equivalent {
		if !listed[m] && slices.Contains(pkgs, path.Dir(m.File)) {
			s.Stale = append(s.Stale, m)
		}
	}
	for _, c := range counts {
		s.Counts = append(s.Counts, *c)
	}
	slices.SortFunc(s.Counts, func(a, b Counts) int { return strings.Compare(a.Package, b.Package) })
	for _, ms := range [][]Mutant{s.Survivors, s.NotCovered, s.Stale} {
		slices.SortFunc(ms, compare)
	}
	return s
}

func compare(a, b Mutant) int {
	if c := strings.Compare(a.File, b.File); c != 0 {
		return c
	}
	if a.Line != b.Line {
		return a.Line - b.Line
	}
	if a.Column != b.Column {
		return a.Column - b.Column
	}
	return strings.Compare(a.Mutator, b.Mutator)
}

// Print writes the survivors one per line, then the not-covered and
// stale mutants under their own headings, then the counts per package.
// An empty section is left out.
func (s Summary) Print(w io.Writer) error {
	b := bufio.NewWriter(w)
	section := func(title string, ms []Mutant) {
		if len(ms) == 0 {
			return
		}
		fmt.Fprintf(b, "%s:\n", title)
		for _, m := range ms {
			fmt.Fprintln(b, m)
		}
		fmt.Fprintln(b)
	}
	section("Survivors", s.Survivors)
	section("Not covered", s.NotCovered)
	section("Stale in mutation/equivalent.txt", s.Stale)

	tw := tabwriter.NewWriter(b, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "package\tkilled\tlived\tequivalent\ttimed out\tnot covered\tnot viable")
	for _, c := range s.Counts {
		fmt.Fprintf(tw, "%s\t%d\t%d\t%d\t%d\t%d\t%d\n",
			c.Package, c.Killed, c.Lived, c.Equivalent, c.TimedOut, c.NotCovered, c.NotViable)
	}
	if err := tw.Flush(); err != nil {
		return err
	}
	return b.Flush()
}
