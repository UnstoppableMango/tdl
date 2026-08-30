package gen

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/unstoppablemango/tdl/plugin"
)

// Write puts a response's files under out.
//
// A backend returns contents rather than writing them, which is what lets
// this enforce where they land. A path is relative to out, and an absolute
// one or one climbing out with ".." is refused before anything is written:
// a plugin cannot reach outside the directory the project pointed it at,
// whatever it intends.
func Write(out string, files []*plugin.File) ([]string, error) {
	cleaned := make([]string, len(files))
	for i, f := range files {
		path, err := resolve(out, f.GetPath())
		if err != nil {
			return nil, err
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

// resolve turns a response path into a real one, or reports why it cannot.
//
// Every path is checked before any file is written, so a response with one
// bad path writes nothing rather than half of itself.
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
