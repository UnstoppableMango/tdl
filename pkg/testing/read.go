package testing

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/spf13/afero"
)

// ReadRawTest attempts to read a valid Test from root.
// A valid test is defined as a directory that contains
// an input.* file and a output.* file.
func ReadRawTest(fsys afero.Fs, root string) (*RawTest, error) {
	var test RawTest

	log.Debug("reading test", "root", root)
	err := afero.Walk(fsys, root,
		func(path string, info fs.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() && path == root {
				test.Name = info.Name()
			}
			if !info.IsDir() && filepath.Dir(path) == root {
				return readTestData(fsys, path, &test)
			}

			return nil
		},
	)
	if err != nil {
		return nil, fmt.Errorf("walking test directory: %w", err)
	}

	if test.Name == "" || test.Input == nil || test.Output == nil {
		return nil, fmt.Errorf("unable to read test: %s", root)
	}

	return &test, nil
}

func readTestData(fsys afero.Fs, path string, test *RawTest) error {
	data, err := afero.ReadFile(fsys, path)
	if err != nil {
		return fmt.Errorf("reading test input: %w", err)
	}

	name := filepath.Base(path)
	if strings.Contains(name, "input") || strings.Contains(name, "source") {
		log.Debug("read input data", "len", len(data))
		test.Input = data
		return nil
	}
	if strings.Contains(name, "output") || strings.Contains(name, "target") {
		log.Debug("read output data", "len", len(data))
		test.Output = data
		return nil
	}

	return fmt.Errorf("invalid test data: %s", path)
}
