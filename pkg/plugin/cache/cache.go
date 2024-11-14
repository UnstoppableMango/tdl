package cache

import (
	"context"
	"io"

	"github.com/adrg/xdg"
	"github.com/spf13/afero"
)

type Cacher interface {
	WriteAll(string, io.Reader) error
	Reader(string) (io.Reader, error)
}

type Cachable interface {
	Cached(Cacher) bool
	Cache(context.Context, Cacher) error
}

var XdgBinHome = &directory{
	fs:   afero.NewOsFs(),
	path: xdg.BinHome,
}
