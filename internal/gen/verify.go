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

// Verify compares a response against what is on disk without writing, and
// returns the paths it would write. The backend still produces full
// contents on a dry run. [Orphaned] finds what nothing would write.
func Verify(out string, files []*plugin.File) ([]Stale, []string, error) {
	var stale []Stale
	var paths []string
	for _, f := range files {
		path, err := resolve(out, f.GetPath())
		if err != nil {
			return nil, nil, err
		}
		paths = append(paths, path)

		switch got, err := os.ReadFile(path); {
		case errors.Is(err, os.ErrNotExist):
			stale = append(stale, Stale{Path: path, Reason: "missing"})
		case err != nil:
			return nil, nil, fmt.Errorf("reading %s: %w", path, err)
		case !bytes.Equal(got, f.GetContent()):
			stale = append(stale, Stale{Path: path, Reason: "differs"})
		}
	}

	return stale, paths, nil
}

// Orphaned lists the files the marker in out lists that are still there and
// that nothing would write. An unlisted file is not tdl's.
func Orphaned(out string, expected map[string]bool) ([]Stale, error) {
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
