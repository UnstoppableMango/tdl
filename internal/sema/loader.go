package sema

import (
	"os"
	"path/filepath"
)

// Loader reads the source of an imported file.
//
// Lowering does not touch the filesystem itself, so a caller can supply
// dependency roots, an in-memory tree, or a vendored directory without
// lowering knowing about any of it.
type Loader interface {
	// Load returns the resolved name of the file imported as path from the
	// file at from, and its source.
	Load(from, path string) (name, src string, err error)
}

// FSLoader resolves an import as a filesystem path relative to the file
// that imports it, which is what the spec says a bare path means.
type FSLoader struct{}

func (FSLoader) Load(from, path string) (string, string, error) {
	name := filepath.Join(filepath.Dir(from), path)
	src, err := os.ReadFile(name)
	if err != nil {
		return name, "", err
	}
	return name, string(src), nil
}
