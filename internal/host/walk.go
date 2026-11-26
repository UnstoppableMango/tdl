package host

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/unstoppablemango/tdl/plugin"
)

// ImportFiles walks dir for the files an importer declaring patterns
// reads, for a directory argument to `tdl import`. It descends into every
// directory but one whose name starts with `.`, and returns the matching
// files sorted, each spelled as dir joined with its path under it, as a
// file named directly would be. A directory yielding no file is an error
// naming it, and so is a backend declaring no patterns.
func ImportFiles(backend, dir string, patterns []string) ([]string, error) {
	if len(patterns) == 0 {
		return nil, fmt.Errorf("%s is a directory, and %s names no files to read from one; name the files instead", dir, backend)
	}

	var files []string
	err := fs.WalkDir(os.DirFS(dir), ".", func(rel string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if rel != "." && strings.HasPrefix(d.Name(), ".") {
				return fs.SkipDir
			}
			return nil
		}
		if d.Type().IsRegular() && plugin.MatchImport(patterns, d.Name()) {
			files = append(files, filepath.Join(dir, filepath.FromSlash(rel)))
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("%s holds no file %s reads (%s)", dir, backend, strings.Join(patterns, ", "))
	}
	slices.Sort(files)
	return files, nil
}
