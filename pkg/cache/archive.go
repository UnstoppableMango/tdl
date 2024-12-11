package cache

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"io/fs"
	"path/filepath"

	"github.com/charmbracelet/log"
	"github.com/spf13/afero"
	"github.com/spf13/afero/tarfs"
	tdl "github.com/unstoppablemango/tdl/pkg"
)

func ExtractTar(bins tdl.Cache, key string, r io.Reader) (err error) {
	gz, err := gzip.NewReader(r)
	if err != nil {
		return fmt.Errorf("reading archive: %w", err)
	}

	defer gz.Close()

	tar := tarfs.New(tar.NewReader(gz))
	return afero.Walk(tar, "",
		func(path string, info fs.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if path == "" || info.IsDir() {
				return nil
			}

			name := filepath.Base(path)
			if Exists(bins, name) {
				log.Debugf("bin exists: %s", name)
				return nil
			}

			if e, err := tar.Open(path); err != nil {
				return fmt.Errorf("opening tar entry: %w", err)
			} else {
				log.Debugf("writing bin: %s", name)
				return WriteAll(bins, name, e)
			}
		},
	)
}
