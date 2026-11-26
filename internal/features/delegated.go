package features

import "regexp"

// Depends is every Generate cell of a delegated backend's table.
const Depends = "Depends on the model"

// delegated are the targets whose mapping a language model answers, such
// as yolo. Measuring one would grade a single recorded answer, so its table
// is written by hand, checked by [CheckDelegated], and read by the matrices
// like a measured one.
var delegated = []string{"yolo"}

// CheckDelegated checks a delegated backend's table: a row per feature in
// table order, under [Header], every Generate cell [Depends], and every
// Import cell No, since a delegated backend does not import.
func CheckDelegated(rows [][]string) error {
	return checkCells(rows, map[int]rule{
		1: {regexp.MustCompile(`^` + regexp.QuoteMeta(Depends) + `$`), Depends},
		2: {regexp.MustCompile(`^No$`), "No"},
	})
}
