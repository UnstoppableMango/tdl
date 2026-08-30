package gen

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/unstoppablemango/tdl/plugin"
)

// Stale describes one way an output directory disagrees with what a
// backend would write.
type Stale struct {
	Path   string
	Reason string
}

// Verify compares a response against what is on disk without writing.
//
// The backend still produces contents: a dry run is about not writing,
// not about producing less. That is what makes the check meaningful, since
// the only way to know whether output is stale is to generate it.
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

// orphaned lists files under a directory tdl owns that this generation
// would not write.
//
// It only looks in a directory carrying the marker. Without one there is
// no way to tell a file tdl wrote and no longer would from a file that was
// never tdl's, and reporting the second as stale would be wrong.
func orphaned(out string, expected map[string]bool) ([]Stale, error) {
	if !Owned(out) {
		return nil, nil
	}

	var stale []Stale
	err := filepath.WalkDir(out, func(path string, d os.DirEntry, err error) error {
		switch {
		case err != nil:
			return err
		case d.IsDir(), filepath.Base(path) == MarkerName, expected[path]:
			return nil
		}
		stale = append(stale, Stale{Path: path, Reason: "no longer generated"})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walking %s: %w", out, err)
	}
	return stale, nil
}
