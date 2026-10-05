package gen

import (
	"bytes"
	"errors"
	"fmt"
	"maps"
	"os"
	"slices"

	"github.com/unstoppablemango/tdl/plugin"
)

// Stale describes one way an output directory disagrees with what a
// backend would write.
type Stale struct {
	Path   string
	Reason string
}

// Verify compares a response against what is on disk without writing.
// The backend still produces full contents on a dry run.
func Verify(out string, files []*plugin.File) ([]Stale, error) {
	var stale []Stale

	expected := map[string]bool{}
	for _, f := range files {
		path, err := resolve(out, f.GetPath())
		if err != nil {
			return nil, err
		}
		expected[path] = true

		switch got, err := os.ReadFile(path); {
		case errors.Is(err, os.ErrNotExist):
			stale = append(stale, Stale{Path: path, Reason: "missing"})
		case err != nil:
			return nil, fmt.Errorf("reading %s: %w", path, err)
		case !bytes.Equal(got, f.GetContent()):
			stale = append(stale, Stale{Path: path, Reason: "differs"})
		}
	}

	orphans, err := orphaned(out, expected)
	if err != nil {
		return nil, err
	}
	return append(stale, orphans...), nil
}

// orphaned lists the files the marker lists that this generation would not
// write and that are still there. An unlisted file is not tdl's.
func orphaned(out string, expected map[string]bool) ([]Stale, error) {
	owned, err := Owned(out)
	if err != nil {
		return nil, err
	}

	var stale []Stale
	for _, path := range slices.Sorted(maps.Keys(owned)) {
		if expected[path] {
			continue
		}
		if _, err := os.Stat(path); err == nil {
			stale = append(stale, Stale{Path: path, Reason: "no longer generated"})
		}
	}
	return stale, nil
}
