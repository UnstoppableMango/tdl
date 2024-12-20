package internal

import (
	"fmt"
	"path/filepath"

	"github.com/charmbracelet/log"
	"github.com/spf13/afero"
	aferox "github.com/unmango/go/fs"
)

func CopyOutput(fs afero.Fs, path string) error {
	path, err := filepath.Abs(path)
	if err != nil {
		return err
	}

	os := afero.NewOsFs()
	dir, err := afero.IsDir(os, path)
	if err != nil {
		return err
	}
	if !dir {
		return fmt.Errorf("not a directory: %s", path)
	}

	log.Debugf("copying output to %s", path)
	return aferox.Copy(fs,
		afero.NewBasePathFs(os, path),
	)
}
