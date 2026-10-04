package gen

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// MarkerName is the file tdl drops in a directory it writes to, so
// --clean knows the directory is its own.
const MarkerName = ".tdl-output"

const markerContent = "This directory is written by `tdl gen`.\n" +
	"`tdl gen --clean` will delete its contents.\n"

// ErrNotOurs is returned when --clean is asked to empty a directory that
// has files but no marker.
var ErrNotOurs = errors.New("the output directory was not written by tdl")

// Mark writes the ownership marker into out.
func Mark(out string) error {
	if err := os.MkdirAll(out, 0o755); err != nil {
		return fmt.Errorf("creating %s: %w", out, err)
	}
	return os.WriteFile(filepath.Join(out, MarkerName), []byte(markerContent), 0o644)
}

// Owned reports whether out carries the marker.
func Owned(out string) bool {
	_, err := os.Stat(filepath.Join(out, MarkerName))
	return err == nil
}

// Clean empties an output directory tdl owns, leaving the marker. A
// missing or empty directory is fine; one with contents and no marker is
// [ErrNotOurs].
func Clean(out string) ([]string, error) {
	entries, err := os.ReadDir(out)
	switch {
	case errors.Is(err, os.ErrNotExist):
		return nil, nil
	case err != nil:
		return nil, fmt.Errorf("reading %s: %w", out, err)
	}

	if len(entries) > 0 && !Owned(out) {
		return nil, fmt.Errorf("%w: %s has no %s", ErrNotOurs, out, MarkerName)
	}

	var removed []string
	for _, e := range entries {
		if e.Name() == MarkerName {
			continue
		}
		path := filepath.Join(out, e.Name())
		if err := os.RemoveAll(path); err != nil {
			return removed, fmt.Errorf("removing %s: %w", path, err)
		}
		removed = append(removed, path)
	}
	return removed, nil
}
