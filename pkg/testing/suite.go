package testing

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/spf13/afero"
	"github.com/unmango/go/option"
)

type Test struct {
	Name   string
	Input  []byte
	Output []byte
}

type discoverOptions struct {
	Depth int
}

type discoverOption func(*discoverOptions)

func Discover(fsys afero.Fs, path string, options ...discoverOption) ([]*Test, error) {
	opts := &discoverOptions{Depth: 2}
	option.ApplyAll(opts, options)

	tests := []*Test{}
	err := afero.Walk(fsys, path,
		func(path string, info fs.FileInfo, err error) error {
			if err != nil {
				return err
			}
			
			parts := strings.Split(path, string(filepath.Separator))

			return nil
		},
	)
	if err != nil {
		return nil, fmt.Errorf("walking filesystem: %w", err)
	}

	return tests, nil
}
