package gen

import (
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// MarkerName is the file tdl drops in a directory it writes to, listing
// the files it wrote there. A file it does not list is someone else's, and
// tdl neither overwrites nor removes it.
const MarkerName = ".tdl-output"

const markerHeader = "# Written by `tdl gen`, which owns the files listed below.\n" +
	"# `tdl gen --clean` removes them; anything else here is left alone.\n"

// legacyMarker is the content of a marker that claimed its whole directory
// rather than listing files, so every file present is read as listed.
const legacyMarker = "This directory is written by `tdl gen`.\n" +
	"`tdl gen --clean` will delete its contents.\n"

// Owned returns the files the marker in out lists, as paths joined to out.
// A directory with no marker owns nothing.
func Owned(out string) (map[string]bool, error) {
	content, err := os.ReadFile(filepath.Join(out, MarkerName))
	switch {
	case errors.Is(err, os.ErrNotExist):
		return map[string]bool{}, nil
	case err != nil:
		return nil, fmt.Errorf("reading %s: %w", filepath.Join(out, MarkerName), err)
	case string(content) == legacyMarker:
		return present(out)
	}

	owned := map[string]bool{}
	for line := range strings.Lines(string(content)) {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			owned[filepath.Join(out, filepath.FromSlash(line))] = true
		}
	}
	return owned, nil
}

// present is every file under out but the marker.
func present(out string) (map[string]bool, error) {
	files := map[string]bool{}
	err := filepath.WalkDir(out, func(path string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() && path != filepath.Join(out, MarkerName) {
			files[path] = true
		}
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("walking %s: %w", out, err)
	}
	return files, nil
}

// Mark adds written to the files the marker in out lists.
func Mark(out string, written []string) error {
	owned, err := Owned(out)
	if err != nil {
		return err
	}
	for _, path := range written {
		owned[path] = true
	}
	return writeMarker(out, owned)
}

func writeMarker(out string, owned map[string]bool) error {
	var b strings.Builder
	b.WriteString(markerHeader)
	for _, path := range slices.Sorted(maps.Keys(owned)) {
		if rel, err := filepath.Rel(out, path); err == nil {
			b.WriteString(filepath.ToSlash(rel) + "\n")
		}
	}
	if err := os.MkdirAll(out, 0o755); err != nil {
		return fmt.Errorf("creating %s: %w", out, err)
	}
	return os.WriteFile(filepath.Join(out, MarkerName), []byte(b.String()), 0o644)
}

// Clean removes the files the marker in out lists, and any directory that
// leaves empty, and empties the list.
func Clean(out string) ([]string, error) {
	owned, err := Owned(out)
	if err != nil || len(owned) == 0 {
		return nil, err
	}

	var removed []string
	for _, path := range slices.Sorted(maps.Keys(owned)) {
		switch err := os.Remove(path); {
		case errors.Is(err, os.ErrNotExist):
			continue
		case err != nil:
			return removed, fmt.Errorf("removing %s: %w", path, err)
		}
		removed = append(removed, path)
		pruneEmpty(out, filepath.Dir(path))
	}
	return removed, writeMarker(out, nil)
}

// pruneEmpty removes dir and each parent below out that removing it
// leaves empty. os.Remove refuses a directory still holding something.
func pruneEmpty(out, dir string) {
	for dir != filepath.Clean(out) && os.Remove(dir) == nil {
		dir = filepath.Dir(dir)
	}
}
