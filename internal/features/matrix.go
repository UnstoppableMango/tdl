package features

import (
	"slices"
	"strings"
)

// The matrices set one column of every target's table side by side: the
// README embeds Generate, and the backends index embeds Import.
const (
	GenerateMatrix = "generate"
	ImportMatrix   = "import"
)

// matrices maps each matrix to the column of [Header] it reads.
var matrices = map[string]int{GenerateMatrix: 1, ImportMatrix: 2}

// Matrix sets column of each named target's table side by side, in the
// order names lists them. A cell is its first word, since the codes a
// Warns names and the phase a Planned names are on the target's own page.
// A target whose column is No throughout, such as one that imports
// nothing, has no column.
func Matrix(column int, names []string, tables map[string]Grid) Grid {
	grid := Grid{Header: []string{"Feature"}}
	var shown []string
	for _, name := range names {
		rows := tables[name].Rows
		if len(rows) != len(Rows) {
			continue
		}
		if slices.ContainsFunc(rows, func(r []string) bool { return word(r[column]) != No.String() }) {
			shown = append(shown, name)
			grid.Header = append(grid.Header, "`"+name+"`")
		}
	}
	for i, row := range Rows {
		cells := []string{row.Title}
		for _, name := range shown {
			cells = append(cells, word(tables[name].Rows[i][column]))
		}
		grid.Rows = append(grid.Rows, cells)
	}
	return grid
}

func word(cell string) string {
	w, _, _ := strings.Cut(cell, " ")
	return w
}
