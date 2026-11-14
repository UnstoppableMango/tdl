package gen

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/unstoppablemango/tdl/plugin"
)

// Write puts a response's files under out. A path is relative to out; an
// absolute one, one leaving out with ".." or through a symlink, or a file
// already there that the marker does not list is refused before anything is
// written.
func Write(out string, files []*plugin.File) ([]string, error) {
	t, err := openTree(out)
	if err != nil {
		return nil, err
	}
	defer t.Close()
	owned, err := t.owned()
	if err != nil {
		return nil, err
	}
	paths := make([]string, len(files))
	rels := make([]string, len(files))
	for i, f := range files {
		path, err := resolve(out, f.GetPath())
		if err != nil {
			return nil, err
		}
		rel, ok := t.rel(path)
		if !ok {
			return nil, escaping(f.GetPath())
		}
		if t.exists(rel) && !owned[path] {
			return nil, fmt.Errorf("%s: tdl did not write this file, so it will not overwrite it", path)
		}
		paths[i], rels[i] = path, rel
	}
	if len(files) > 0 {
		if err := t.create(); err != nil {
			return nil, err
		}
	}

	written := make([]string, 0, len(files))
	for i, f := range files {
		if err := t.root.MkdirAll(filepath.Dir(rels[i]), 0o755); err != nil {
			return written, fmt.Errorf("creating %s: %w", filepath.Dir(paths[i]), err)
		}
		if err := t.root.WriteFile(rels[i], f.GetContent(), 0o644); err != nil {
			return written, fmt.Errorf("writing %s: %w", paths[i], err)
		}
		written = append(written, paths[i])
	}
	return written, nil
}

// resolve turns a response path into one under out, or reports why it
// cannot.
func resolve(out, path string) (string, error) {
	switch {
	case path == "":
		return "", fmt.Errorf("the backend returned a file with no path")
	case filepath.IsAbs(path):
		return "", fmt.Errorf("%s: a backend may not write an absolute path", path)
	}

	full := filepath.Join(out, path)
	rel, err := filepath.Rel(out, full)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", escaping(path)
	}
	return full, nil
}

func escaping(path string) error {
	return fmt.Errorf("%s: a backend may not write outside the output directory", path)
}
