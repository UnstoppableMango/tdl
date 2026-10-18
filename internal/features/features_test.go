package features_test

import (
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/unstoppablemango/tdl/internal/features"
	"github.com/unstoppablemango/tdl/internal/gen/echo"
)

const root = "../.."

// Every committed table is what measuring the backends writes now. A cell
// a tool missing from PATH leaves unmeasured keeps its committed text.
func TestTablesAreCurrent(t *testing.T) {
	docs, err := features.Update(root, testing.Short())
	if err != nil {
		t.Fatal(err)
	}
	for _, doc := range slices.Sorted(maps.Keys(docs)) {
		committed, err := os.ReadFile(filepath.Join(root, doc))
		if err != nil {
			t.Fatal(err)
		}
		if string(committed) != string(docs[doc]) {
			t.Errorf("%s is stale; run make features and read the diff", doc)
		}
	}
}

// Every row has a source, and every source is a row.
func TestEveryRowHasASource(t *testing.T) {
	var rows []string
	for _, r := range features.Rows {
		rows = append(rows, r.Dir)
		if _, err := os.Stat(filepath.Join(root, features.Corpus, r.Dir, features.Source)); err != nil {
			t.Errorf("%s: %v", r.Dir, err)
		}
	}
	entries, err := os.ReadDir(filepath.Join(root, features.Corpus))
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if e.IsDir() && !slices.Contains(rows, e.Name()) {
			t.Errorf("%s/%s is no row", features.Corpus, e.Name())
		}
	}
}

// A row's codes are loss codes docs/design/reverse.md lists.
func TestRowCodesAreLossCodes(t *testing.T) {
	doc, err := os.ReadFile(filepath.Join(root, "docs/design/reverse.md"))
	if err != nil {
		t.Fatal(err)
	}
	var codes []string
	for _, m := range regexp.MustCompile("(?m)^\\| `(lossy\\.[a-z-]+)` \\|").FindAllSubmatch(doc, -1) {
		codes = append(codes, string(m[1]))
	}
	for _, r := range features.Rows {
		for _, c := range r.Codes {
			if !slices.Contains(codes, c) {
				t.Errorf("%s owns %s, which reverse.md does not list", r.Dir, c)
			}
		}
	}
}

// echo reads back exactly what it wrote, so every row comes back Yes but
// its warning on every import, which no row owns.
func TestMeasureEcho(t *testing.T) {
	measured, err := features.Measure(filepath.Join(root, features.Corpus), features.Target{Name: "echo", Backend: echo.Backend{}})
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range features.Rows {
		m := measured[r.Dir]
		// echo declares no directives, so the rows using one are No.
		want := features.Yes
		if r.Dir == "key" || r.Dir == "name" || r.Dir == "number" {
			want = features.No
		}
		if m.Generate.Support != want {
			t.Errorf("%s generates %v (%s), want %v", r.Dir, m.Generate.Support, m.Generate.Why, want)
		}
	}
}

func TestHand(t *testing.T) {
	hand, err := features.ReadHand([]byte("[min]\nspelling = \"`minimum`\"\nplanned = \"phase 5\"\n"))
	if err != nil {
		t.Fatal(err)
	}
	if h := hand["min"]; h.Spelling != "`minimum`" || h.Planned != "phase 5" {
		t.Errorf("hand = %+v", h)
	}
	for _, bad := range []string{"[nope]\nnotes = \"x\"\n", "[min]\ncolour = \"x\"\n", "[min]\nnotes = \"a | b\"\n"} {
		if _, err := features.ReadHand([]byte(bad)); err == nil {
			t.Errorf("ReadHand(%q) succeeded", bad)
		}
	}
}

// A planned row that already generates is an error, so a stale plan
// cannot outlive the work.
func TestPlannedYesFails(t *testing.T) {
	measured := map[string]features.Measured{"min": {Generate: features.Cell{Support: features.Yes}}}
	if _, err := features.Table(measured, map[string]features.Hand{"min": {Planned: "phase 5"}}); err == nil {
		t.Error("Table succeeded")
	}
	measured["min"] = features.Measured{Generate: features.Cell{Support: features.Warns, Codes: []string{"lossy.constraint"}}}
	rows, err := features.Table(measured, map[string]features.Hand{"min": {Planned: "phase 5"}})
	if err != nil {
		t.Fatal(err)
	}
	if got := rows[slices.IndexFunc(features.Rows, func(r features.Row) bool { return r.Dir == "min" })][1]; got != "Planned (phase 5)" {
		t.Errorf("min generates %q", got)
	}
}

// Splice keeps a committed cell where the new one is unmeasured, and
// fails when there is none to keep.
func TestSpliceKeepsUnmeasured(t *testing.T) {
	doc := "# Doc\n\n<!-- tdl:features x -->\n\n" +
		features.Render(features.Header, [][]string{{"Alias", "Yes", "Warns (`lossy.alias`)", "", ""}}) +
		"\n<!-- /tdl:features -->\n"
	out, err := features.Splice([]byte(doc), map[string][][]string{"x": {{"Alias", "Yes", "Unmeasured", "", ""}}})
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != doc {
		t.Errorf("spliced\n%s\nwant\n%s", out, doc)
	}
	if _, err := features.Splice([]byte(doc), map[string][][]string{"x": {{"Newtype", "Yes", "Unmeasured", "", ""}}}); err == nil || !strings.Contains(err.Error(), "Newtype") {
		t.Errorf("Splice = %v, want an error naming Newtype", err)
	}
}
