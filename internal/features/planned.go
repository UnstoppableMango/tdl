package features

import (
	"errors"
	"fmt"
	"regexp"
)

// planned is a cell a backend with no code may hold: the phase that will
// build the feature, or No when none will.
var planned = regexp.MustCompile(`^(Planned \(.+\)|No)$`)

// CheckPlanned checks the table a backend with no code embeds, which nothing
// measures: a row per feature in table order, under [Header], and no
// Generate or Import cell but a Planned naming a phase or a No. Once the
// backend is built, its table is measured instead.
func CheckPlanned(rows [][]string) error {
	r := rule{planned, "Planned (phase) or No"}
	return checkCells(rows, r, r)
}

// rule is what a column of a table written by hand may hold, and how an
// error describes it.
type rule struct {
	cell *regexp.Regexp
	what string
}

// checkCells checks a table written by hand: a row per feature in table
// order, under [Header], and its Generate and Import cells holding what
// their rules allow.
func checkCells(rows [][]string, generate, imports rule) error {
	if len(rows) != len(Rows) {
		return fmt.Errorf("%d rows, want one per feature, %d", len(rows), len(Rows))
	}
	var problems []error
	for i, r := range rows {
		if len(r) != len(Header) {
			problems = append(problems, fmt.Errorf("row %d has %d cells, want %d", i+1, len(r), len(Header)))
			continue
		}
		if r[0] != Rows[i].Title {
			problems = append(problems, fmt.Errorf("row %d is %q, want %q", i+1, r[0], Rows[i].Title))
		}
		for i, rule := range []rule{generate, imports} {
			if j := i + 1; !rule.cell.MatchString(r[j]) {
				problems = append(problems, fmt.Errorf("%s: %s is %q, want %s", r[0], Header[j], r[j], rule.what))
			}
		}
	}
	return errors.Join(problems...)
}
