package sema

import "os"

// MapLoader resolves imports from an in-memory tree, keyed by the path as
// written, so a test can exercise imports without touching the filesystem.
type MapLoader map[string]string

func (m MapLoader) Load(_, path string) (string, string, error) {
	src, ok := m[path]
	if !ok {
		return path, "", os.ErrNotExist
	}
	return path, src, nil
}
