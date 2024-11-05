package cache

import (
	"io"

	"github.com/adrg/xdg"
	"github.com/spf13/afero"
)

type Cacher interface {
	Write(string, []byte) error
	Reader(string) (io.Reader, error)
}

var XdgBinHome = &directory{
	fs:   afero.NewOsFs(),
	path: xdg.BinHome,
}

func WriteAll(cache Cacher, name string, reader io.Reader) error {
	data, err := io.ReadAll(reader)
	if err != nil {
		return err
	}

	return cache.Write(name, data)
}
