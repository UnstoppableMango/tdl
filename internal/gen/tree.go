package gen

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// tree is an output directory, opened as an [os.Root] so that nothing tdl
// reads, writes, or removes under it leaves it, through a symlink or
// otherwise. A directory that is itself a symlink opens at its target; a
// symlink inside it pointing out of it is refused.
type tree struct {
	dir  string
	root *os.Root // nil while dir does not exist
}

func openTree(dir string) (*tree, error) {
	t := &tree{dir: filepath.Clean(dir)}
	root, err := os.OpenRoot(t.dir)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return t, nil
	case err != nil:
		return nil, fmt.Errorf("opening %s: %w", dir, err)
	}
	t.root = root
	return t, nil
}

// create makes the directory if it does not exist yet.
func (t *tree) create() error {
	if t.root != nil {
		return nil
	}
	if err := os.MkdirAll(t.dir, 0o755); err != nil {
		return fmt.Errorf("creating %s: %w", t.dir, err)
	}
	root, err := os.OpenRoot(t.dir)
	if err != nil {
		return fmt.Errorf("opening %s: %w", t.dir, err)
	}
	t.root = root
	return nil
}

func (t *tree) Close() {
	if t.root != nil {
		t.root.Close()
	}
}

// rel turns a path joined to the directory into one relative to it, and
// reports false for a path that leaves it, by its spelling or through a
// symlink.
func (t *tree) rel(path string) (string, bool) {
	rel, err := filepath.Rel(t.dir, path)
	if err != nil || !filepath.IsLocal(rel) {
		return "", false
	}
	return rel, t.root == nil || !escapes(t.root, rel)
}

// escapes reports whether rel passes through a symlink that resolves outside
// root. The root refuses such a path itself; asking first gives a clear
// error, and lets a write refuse before it writes anything.
func escapes(root *os.Root, rel string) bool {
	prefix := ""
	for part := range strings.SplitSeq(rel, string(filepath.Separator)) {
		prefix = filepath.Join(prefix, part)
		info, err := root.Lstat(prefix)
		if err != nil {
			// Nothing further exists, or the operation itself reports why.
			return false
		}
		if info.Mode()&fs.ModeSymlink == 0 {
			continue
		}
		if _, err := root.Stat(prefix); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return true
		}
	}
	return false
}

func (t *tree) readFile(rel string) ([]byte, error) {
	if t.root == nil {
		return nil, &fs.PathError{Op: "open", Path: filepath.Join(t.dir, rel), Err: fs.ErrNotExist}
	}
	return t.root.ReadFile(rel)
}

// exists reports whether something is at rel.
func (t *tree) exists(rel string) bool {
	if t.root == nil {
		return false
	}
	_, err := t.root.Stat(rel)
	return err == nil
}

// local is rel for a path tdl is about to read or remove, refusing one
// that leaves the directory with an error naming the path as the marker
// spells it; verb says what tdl would have done.
func (t *tree) local(path, verb string) (string, error) {
	rel, ok := t.rel(path)
	if !ok {
		name := path
		if r, err := filepath.Rel(t.dir, path); err == nil {
			name = filepath.ToSlash(r)
		}
		return "", fmt.Errorf("%s: tdl will not %s a file outside the output directory", name, verb)
	}
	return rel, nil
}
