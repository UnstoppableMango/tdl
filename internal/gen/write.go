package gen

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/unstoppablemango/tdl/plugin"
)

// Write puts a response's files under out. A path is relative to out; an
// absolute one, one climbing out with "..", or a file already there that
// the marker does not list is refused before anything is written.
func Write(out string, files []*plugin.File) ([]string, error) {
	owned, err := Owned(out)
	if err != nil {
		return nil, err
	}
	cleaned := make([]string, len(files))
	for i, f := range files {
		path, err := resolve(out, f.GetPath())
		if err != nil {
			return nil, err
		}
		if _, err := os.Stat(path); err == nil && !owned[path] {
			return nil, fmt.Errorf("%s: tdl did not write this file, so it will not overwrite it", path)
		}
		cleaned[i] = path
	}

	written := make([]string, 0, len(files))
	for i, f := range files {
		if err := os.MkdirAll(filepath.Dir(cleaned[i]), 0o755); err != nil {
			return written, fmt.Errorf("creating %s: %w", filepath.Dir(cleaned[i]), err)
		}
		if err := os.WriteFile(cleaned[i], f.GetContent(), 0o644); err != nil {
			return written, fmt.Errorf("writing %s: %w", cleaned[i], err)
		}
		written = append(written, cleaned[i])
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
		return "", fmt.Errorf("%s: a backend may not write outside the output directory", path)
	}
	return full, nil
}
