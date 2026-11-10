package gen

import (
	"errors"
	"fmt"
	"io/fs"
	"maps"
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
	t, err := openTree(out)
	if err != nil {
		return nil, err
	}
	defer t.Close()
	return t.owned()
}

func (t *tree) owned() (map[string]bool, error) {
	content, err := t.readFile(MarkerName)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return map[string]bool{}, nil
	case err != nil:
		return nil, fmt.Errorf("reading %s: %w", filepath.Join(t.dir, MarkerName), err)
	case string(content) == legacyMarker:
		return t.present()
	}

	owned := map[string]bool{}
	for line := range strings.Lines(string(content)) {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			owned[filepath.Join(t.dir, filepath.FromSlash(line))] = true
		}
	}
	return owned, nil
}

// present is every file under the directory but the marker.
func (t *tree) present() (map[string]bool, error) {
	files := map[string]bool{}
	err := fs.WalkDir(t.root.FS(), ".", func(path string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() && path != MarkerName {
			files[filepath.Join(t.dir, filepath.FromSlash(path))] = true
		}
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("walking %s: %w", t.dir, err)
	}
	return files, nil
}

// Mark adds written to the files the marker in out lists.
func Mark(out string, written []string) error {
	t, err := openTree(out)
	if err != nil {
		return err
	}
	defer t.Close()
	owned, err := t.owned()
	if err != nil {
		return err
	}
	for _, path := range written {
		owned[path] = true
	}
	kept, err := t.readKept()
	if err != nil {
		return err
	}
	return t.writeMarker(owned, kept)
}

func (t *tree) writeMarker(owned map[string]bool, kept map[string]Kept) error {
	var b strings.Builder
	b.WriteString(markerHeader)
	for _, path := range slices.Sorted(maps.Keys(owned)) {
		if rel, err := filepath.Rel(t.dir, path); err == nil {
			b.WriteString(filepath.ToSlash(rel) + "\n")
		}
	}
	t.writeKept(&b, owned, kept)
	if err := t.create(); err != nil {
		return err
	}
	return t.root.WriteFile(MarkerName, []byte(b.String()), 0o644)
}

// Clean removes the files the marker in out lists, and any directory that
// leaves empty, and empties the list. A listed path leaving out through a
// symlink is not followed: it is reported and stays listed, so the next
// clean reports it again, and the rest are still removed.
func Clean(out string) ([]string, error) {
	t, err := openTree(out)
	if err != nil {
		return nil, err
	}
	defer t.Close()
	owned, err := t.owned()
	if err != nil || len(owned) == 0 {
		return nil, err
	}

	var removed []string
	var refused []error
	kept := map[string]bool{}
	for _, path := range slices.Sorted(maps.Keys(owned)) {
		rel, err := t.local(path, "remove")
		if err != nil {
			refused = append(refused, err)
			kept[path] = true
			continue
		}
		switch err := t.root.Remove(rel); {
		case errors.Is(err, fs.ErrNotExist):
			continue
		case err != nil:
			return removed, errors.Join(append(refused, fmt.Errorf("removing %s: %w", path, err))...)
		}
		removed = append(removed, path)
		t.pruneEmpty(filepath.Dir(rel))
	}
	return removed, errors.Join(append(refused, t.writeMarker(kept, nil))...)
}

// pruneEmpty removes dir and each parent below the directory that removing
// it leaves empty. Removing refuses a directory still holding something, and
// a symlink is never removed, since the directory it names is not emptied.
func (t *tree) pruneEmpty(dir string) {
	for dir != "." {
		if info, err := t.root.Lstat(dir); err != nil || !info.IsDir() || t.root.Remove(dir) != nil {
			return
		}
		dir = filepath.Dir(dir)
	}
}
