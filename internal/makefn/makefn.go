// Package makefn computes what the GNU make functions cmd/tdl-gmk defines
// return, apart from make, so they are tested without loading a shared
// object.
package makefn

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"

	"github.com/unstoppablemango/tdl/parser"
)

// Ext is the extension of a TDL source file.
const Ext = ".tdl"

// Files returns the TDL files under each of dirs, sorted, as slash-separated
// paths relative to root. A relative dir is relative to root. A dir that does
// not exist lists nothing, as $(wildcard) would; any other error is returned
// rather than part of a tree.
func Files(root string, dirs []string) ([]string, error) {
	var files []string
	for _, dir := range dirs {
		dir = abs(root, dir)
		if _, err := os.Lstat(dir); errors.Is(err, fs.ErrNotExist) {
			continue
		}
		err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() || filepath.Ext(path) != Ext {
				return err
			}
			files = append(files, path)
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	return relative(root, files)
}

// Imports returns files and every file they import, transitively, sorted and
// relative to root as Files returns them. An import resolves against the
// directory of the file writing it, as tdl resolves it.
func Imports(root string, files []string) ([]string, error) {
	seen := map[string]bool{}
	queue := make([]string, 0, len(files))
	for _, f := range files {
		queue = append(queue, abs(root, f))
	}
	for len(queue) > 0 {
		name := queue[0]
		queue = queue[1:]
		if seen[name] {
			continue
		}
		seen[name] = true
		src, err := os.ReadFile(name)
		if err != nil {
			return nil, err
		}
		file, err := parser.Parse(name, bytes.NewReader(src))
		if err != nil {
			return nil, err
		}
		for _, imp := range file.Imports {
			queue = append(queue, filepath.Join(filepath.Dir(name), imp.Path))
		}
	}
	all := make([]string, 0, len(seen))
	for name := range seen {
		all = append(all, name)
	}
	return relative(root, all)
}

func abs(root, path string) string {
	if filepath.IsAbs(path) {
		return filepath.Clean(path)
	}
	return filepath.Join(root, path)
}

func relative(root string, paths []string) ([]string, error) {
	out := make([]string, 0, len(paths))
	for _, p := range paths {
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", p, err)
		}
		out = append(out, filepath.ToSlash(rel))
	}
	slices.Sort(out)
	return slices.Compact(out), nil
}
