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
// target and matrix is embedded exactly once, and every table names one.
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
	slices.Sort(names)
	for _, name := range sortedKeys(matrices) {
		problems = append(problems, once(name, where[name])...)
		delete(where, name)
		tables[name] = Matrix(matrices[name], names, tables)
	}
	for _, name := range sortedKeys(where) {
		problems = append(problems, fmt.Errorf("%s embed a table for %s, which is no target or matrix", strings.Join(where[name], ", "), name))
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
