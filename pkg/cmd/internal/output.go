package internal

import (
	"os"
	"path/filepath"

	"github.com/charmbracelet/log"
	"github.com/spf13/afero"
	aferox "github.com/unmango/aferox"
)

func CopyOutput(fs afero.Fs, path string) error {
	path, err := filepath.Abs(path)
	if err != nil {
		return err
	}

	outfs := afero.NewOsFs()
	exists, err := afero.DirExists(outfs, path)
	if err != nil {
		return err
	}
	if !exists {
		if err = outfs.Mkdir(path, os.ModePerm); err != nil {
			return err
		}
	}

	log.Debugf("copying output to %s", path)
	return aferox.Copy(fs,
		afero.NewBasePathFs(outfs, path),
	)
}
