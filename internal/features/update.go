package features

import (
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// Update measures every target and returns, for each markdown file under
// root embedding a table, its content with the tables rewritten. Every
// target, delegated backend, and matrix is embedded exactly once. A
// delegated backend's table is checked by [CheckDelegated] and left as it
// is, and a table naming none of them is a planned backend's, which is
// checked by [CheckPlanned] and left as it is.
//
// Unless short, a tool or plugin missing from PATH is an error, so a
// committed cell never stands in for one that could not be measured; short
// leaves those cells unmeasured, and they keep their committed text.
func Update(root string, short bool) (map[string][]byte, error) {
	embedded, err := embedding(root)
	if err != nil {
		return nil, err
	}

	var problems []error
	where := map[string][]string{}
	for doc, data := range embedded {
		for _, name := range Markers(data) {
			where[name] = append(where[name], doc)
		}
	}

	tables := map[string]Grid{}
	var names []string
	for _, t := range Targets(short) {
		if err := t.missing(); !short && err != nil {
			problems = append(problems, fmt.Errorf("%s: %w; -short leaves its cells unmeasured", t.Name, err))
		}
		docs := where[t.Name]
		problems = append(problems, once(t.Name, docs)...)
		delete(where, t.Name)

		rows, err := table(root, t)
		if err != nil {
			problems = append(problems, fmt.Errorf("%s: %w", t.Name, err))
			continue
		}
		grid := Grid{Header: Header, Rows: rows}
		// Keep committed cells here rather than in Splice, so the
		// matrices read them too.
		if len(docs) == 1 {
			if grid.Rows, err = keep(grid, Parse(embedded[docs[0]], t.Name)); err != nil {
				problems = append(problems, fmt.Errorf("%s: %w", docs[0], err))
			}
		}
		tables[t.Name] = grid
		names = append(names, t.Name)
	}
	for _, name := range delegated {
		rows, errs := written(embedded, where[name], name, "a delegated backend", CheckDelegated)
		delete(where, name)
		if errs != nil {
			problems = append(problems, errs...)
			continue
		}
		tables[name] = Grid{Header: Header, Rows: rows}
		names = append(names, name)
	}
	slices.Sort(names)
	for _, name := range sortedKeys(matrices) {
		problems = append(problems, once(name, where[name])...)
		delete(where, name)
		tables[name] = Matrix(matrices[name], names, tables)
	}
	// What remains is a backend with no code, whose table is written by
	// hand and only checked.
	for _, name := range sortedKeys(where) {
		_, errs := written(embedded, where[name], name, "no target, so a planned table", CheckPlanned)
		problems = append(problems, errs...)
	}

	out := map[string][]byte{}
	for _, doc := range sortedKeys(embedded) {
		spliced, err := Splice(embedded[doc], tables)
		if err != nil {
			problems = append(problems, fmt.Errorf("%s: %w", doc, err))
		}
		out[doc] = spliced
	}
	return out, errors.Join(problems...)
}

// written reads the table name that one of docs embeds, which is written
// by hand, and checks it; kind says what the table is in an error.
func written(embedded map[string][]byte, docs []string, name, kind string, check func([][]string) error) ([][]string, []error) {
	if errs := once(name, docs); errs != nil {
		return nil, errs
	}
	rows := Parse(embedded[docs[0]], name)
	if err := check(rows); err != nil {
		return nil, []error{fmt.Errorf("%s: %s is %s: %w", docs[0], name, kind, err)}
	}
	return rows, nil
}

// once reports a table embedded in any number of docs but one.
func once(name string, docs []string) []error {
	switch len(docs) {
	case 0:
		return []error{fmt.Errorf("no document embeds the %s table", name)}
	case 1:
		return nil
	default:
		slices.Sort(docs)
		return []error{fmt.Errorf("%s embed the %s table; one may", strings.Join(docs, ", "), name)}
	}
}

func table(root string, t Target) ([][]string, error) {
	measured, err := Measure(filepath.Join(root, Corpus), t)
	if err != nil {
		return nil, err
	}
	hand := map[string]Hand{}
	file := filepath.Join(root, t.Hand)
	data, err := os.ReadFile(file)
	switch {
	case errors.Is(err, fs.ErrNotExist):
	case err != nil:
		return nil, err
	default:
		if hand, err = ReadHand(data); err != nil {
			return nil, fmt.Errorf("%s: %w", file, err)
		}
	}
	return Table(measured, hand)
}

// embedding reads every markdown file under root that embeds a table,
// keyed by its path relative to root.
func embedding(root string) (map[string][]byte, error) {
	docs := map[string][]byte{}
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if name := d.Name(); p != root && (strings.HasPrefix(name, ".") || name == "node_modules" || name == "testdata") {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(p) != ".md" {
			return nil
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		if len(Markers(data)) > 0 {
			rel, err := filepath.Rel(root, p)
			if err != nil {
				return err
			}
			docs[rel] = data
		}
		return nil
	})
	return docs, err
}

func sortedKeys[V any](m map[string]V) []string {
	return slices.Sorted(maps.Keys(m))
}
