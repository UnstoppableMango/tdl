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

func StoreTarGz(bins tdl.Cache, r io.Reader) error {
	if gz, err := gzip.NewReader(r); err != nil {
		return err
	} else {
		return StoreTar(bins, gz)
	}
}

func StoreTar(bins tdl.Cache, r io.Reader) error {
	return StoreTarFs(bins,
		tarfs.New(tar.NewReader(r)),
	)
}

func StoreTarFs(bins tdl.Cache, tar *tarfs.Fs) (err error) {
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
