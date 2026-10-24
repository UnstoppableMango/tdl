package sema

import (
	"os"
	"path/filepath"
)

// Loader reads the source of an imported file. Lowering does not touch the
// filesystem itself.
type Loader interface {
	// Load returns the resolved name of the file imported as path from the
	// file at from, and its source.
	Load(from, path string) (name, src string, err error)
}

// FSLoader resolves an import as a filesystem path relative to the file
// that imports it.
type FSLoader struct{}

func (FSLoader) Load(from, path string) (string, string, error) {
	name := filepath.Join(filepath.Dir(from), path)
	src, err := os.ReadFile(name)
	if err != nil {
		return name, "", err
	}
	return name, string(src), nil
}

// Overlay resolves an import as FSLoader does, and reads the resolved
// file's text from Text when it holds one, such as an editor's unsaved
// buffer, and from disk otherwise.
type Overlay struct {
	// Text returns the text the overlay holds for a resolved name.
	Text func(name string) (string, bool)
}

func (o Overlay) Load(from, path string) (string, string, error) {
	name := filepath.Join(filepath.Dir(from), path)
	if src, ok := o.Text(name); ok {
		return name, src, nil
	}
	return FSLoader{}.Load(from, path)
}
