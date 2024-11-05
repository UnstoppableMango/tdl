package cache

import (
	"archive/tar"
	"compress/gzip"
	"errors"
	"io"
	"slices"

	"github.com/charmbracelet/log"
)

func TarGz(cache Cacher, reader io.Reader, files ...string) error {
	gzip, err := gzip.NewReader(reader)
	if err != nil {
		return err
	}

	return Tar(cache, gzip, files...)
}

func Tar(cache Cacher, reader io.Reader, files ...string) error {
	return TarEntries(cache, tar.NewReader(reader), files)
}

func TarEntries(cache Cacher, reader *tar.Reader, files []string) error {
	if len(files) == 0 {
		return errors.New("no files to cache")
	}

	for {
		header, err := reader.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		name := header.Name
		if !slices.Contains(files, name) {
			log.Debug("skipping tar entry", "name", name)
			continue
		}

		err = All(cache, name, reader)
		if err != nil {
			return err
		}
	}
}
