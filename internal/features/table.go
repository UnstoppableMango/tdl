package features

import (
	"bytes"
	"errors"
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strings"

	"github.com/BurntSushi/toml"
)

// Hand is what a backend's features.toml says about one row: the cells a
// measurement cannot fill.
type Hand struct {
	// Spelling is how the target writes the feature.
	Spelling string `toml:"spelling"`
	// Notes say why a cell is what it is.
	Notes string `toml:"notes"`
	// Planned names the phase that will generate the feature, shown in
	// place of a Warns or No.
	Planned string `toml:"planned"`
}

// ReadHand reads a backend's features.toml, keyed by row directory. A
// cell may not hold a `|`, which would end it.
func ReadHand(data []byte) (map[string]Hand, error) {
	hand := map[string]Hand{}
	md, err := toml.Decode(string(data), &hand)
	if err != nil {
		return nil, err
	}
	var problems []error
	for _, key := range md.Undecoded() {
		problems = append(problems, fmt.Errorf("unknown key %s", key))
	}
	for _, dir := range slices.Sorted(maps.Keys(hand)) {
		if !slices.ContainsFunc(Rows, func(r Row) bool { return r.Dir == dir }) {
			problems = append(problems, fmt.Errorf("[%s] names no feature", dir))
		}
		h := hand[dir]
		if strings.Contains(h.Spelling+h.Notes+h.Planned, "|") {
			problems = append(problems, fmt.Errorf("[%s] holds a |, which ends a table cell", dir))
		}
	}
	return hand, errors.Join(problems...)
}

// Header is a backend table's first row.
var Header = []string{"Feature", "Generate", "Import", "Target spelling", "Notes"}

// unmeasured stands in a cell a measurement skipped until [Splice] takes
// the committed one.
const unmeasured = "Unmeasured"

// Table is a backend's rows, one per feature, in [Header]'s columns. A
// row planned for a phase that already generates it is an error, so a
// stale plan cannot outlive the work.
func Table(measured map[string]Measured, hand map[string]Hand) ([][]string, error) {
	var rows [][]string
	var problems []error
	for _, row := range Rows {
		m, h := measured[row.Dir], hand[row.Dir]
		generate := text(m.Generate)
		if h.Planned != "" {
			if m.Generate.Support == Yes {
				problems = append(problems, fmt.Errorf("%s is planned for %s and already generates", row.Dir, h.Planned))
			}
			generate = "Planned (" + h.Planned + ")"
		}
		rows = append(rows, []string{row.Title, generate, text(m.Import), h.Spelling, h.Notes})
	}
	return rows, errors.Join(problems...)
}

func text(c Cell) string {
	switch c.Support {
	case Unmeasured:
		return unmeasured
	case Warns:
		codes := make([]string, len(c.Codes))
		for i, code := range c.Codes {
			codes[i] = "`" + code + "`"
		}
		return "Warns (" + strings.Join(codes, ", ") + ")"
	default:
		return c.Support.String()
	}
}

// Grid is a table to embed: its header and its rows.
type Grid struct {
	Header []string
	Rows   [][]string
}

// Render writes rows as a markdown table under header.
func Render(header []string, rows [][]string) string {
	var b strings.Builder
	line := func(cells []string) {
		b.WriteString("|")
		for _, c := range cells {
			if c == "" {
				b.WriteString(" |")
			} else {
				b.WriteString(" " + c + " |")
			}
		}
		b.WriteString("\n")
	}
	line(header)
	sep := make([]string, len(header))
	for i := range sep {
		sep[i] = "---"
	}
	line(sep)
	for _, r := range rows {
		line(r)
	}
	return b.String()
}

// marker matches one embedded table: its opening comment naming what it
// holds, the table, and the closing comment.
var marker = regexp.MustCompile(`(?s)<!-- tdl:features ([a-z0-9_]+) -->\n(.*?)<!-- /tdl:features -->`)

// Markers lists the names of the tables a document embeds, in order.
func Markers(doc []byte) []string {
	var names []string
	for _, m := range marker.FindAllSubmatch(doc, -1) {
		names = append(names, string(m[1]))
	}
	return names
}

// Splice replaces the tables a document embeds with those tables names,
// keeping a committed cell wherever the new one is unmeasured. A table
// the document embeds and tables lacks is left alone.
func Splice(doc []byte, tables map[string]Grid) ([]byte, error) {
	var problems []error
	out := marker.ReplaceAllFunc(doc, func(block []byte) []byte {
		m := marker.FindSubmatch(block)
		name := string(m[1])
		grid, ok := tables[name]
		if !ok {
			return block
		}
		rows, err := keep(grid, parse(m[2]))
		if err != nil {
			problems = append(problems, fmt.Errorf("%s: %w", name, err))
		}
		var b bytes.Buffer
		fmt.Fprintf(&b, "<!-- tdl:features %s -->\n\n%s\n<!-- /tdl:features -->", name, Render(grid.Header, rows))
		return b.Bytes()
	})
	return out, errors.Join(problems...)
}

// Parse reads the rows of the table a document embeds under name, without
// its header, or nil when it embeds none.
func Parse(doc []byte, name string) [][]string {
	for _, m := range marker.FindAllSubmatch(doc, -1) {
		if string(m[1]) == name {
			return parse(m[2])
		}
	}
	return nil
}

func parse(table []byte) [][]string {
	var rows [][]string
	for _, line := range strings.Split(string(table), "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "|") {
			continue
		}
		var cells []string
		for _, c := range strings.Split(strings.Trim(line, "|"), "|") {
			cells = append(cells, strings.TrimSpace(c))
		}
		rows = append(rows, cells)
	}
	if len(rows) < 2 {
		return nil
	}
	return rows[2:]
}

// keep fills each unmeasured cell from the committed row with the same
// feature, which must exist.
func keep(grid Grid, committed [][]string) ([][]string, error) {
	byTitle := map[string][]string{}
	for _, r := range committed {
		if len(r) > 0 {
			byTitle[r[0]] = r
		}
	}
	var problems []error
	out := make([][]string, len(grid.Rows))
	for i, r := range grid.Rows {
		out[i] = slices.Clone(r)
		for j, c := range r {
			if c != unmeasured {
				continue
			}
			old, ok := byTitle[r[0]]
			if !ok || j >= len(old) || old[j] == unmeasured {
				problems = append(problems, fmt.Errorf("%s: %s is unmeasured here and has no committed cell", r[0], grid.Header[j]))
				continue
			}
			out[i][j] = old[j]
		}
	}
	return out, errors.Join(problems...)
}
