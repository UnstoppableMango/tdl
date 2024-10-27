package testing

import (
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/spf13/afero"
)

func Discover(fsys afero.Fs, root string) ([]*Test, error) {
	tests := []*Test{}
	log.Debug("walking filesystem", "root", root)
	err := afero.Walk(fsys, root,
		func(path string, info fs.FileInfo, err error) error {
			log := log.With("path", path)
			if err != nil {
				return err
			}
			if !info.IsDir() {
				log.Debug("skipping file")
				return nil
			}

			relative, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			depth := len(strings.Split(relative, string(filepath.Separator)))
			if depth != 1 || relative == "." {
				log.Debug("unsupported depth", "depth", depth, "rel", relative)
				return nil
			}

			test, err := ReadTest(fsys, path)
			if err != nil {
				log.Debug("skipping invalid test", "rel", relative, "err", err)
				return filepath.SkipDir
			}
			if test == nil {
				return errors.New("test was nil, this is a developer mistake")
			}

			log.Debug("appending test", "test", test.Name)
			tests = append(tests, test)
			return nil
		},
	)
	if err != nil {
		return nil, fmt.Errorf("walking filesystem: %w", err)
	}

	return tests, nil
}
