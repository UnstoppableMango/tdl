package cache

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

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

func Fs() (afero.Fs, error) {
	base := afero.NewOsFs()
	path := filepath.Join(xdg.CacheHome, "ux")
	info, err := base.Stat(path)

	if errors.Is(err, os.ErrNotExist) {
		err = base.MkdirAll(path, os.ModeDir)
	}
	if err != nil {
		return nil, fmt.Errorf("creating cache directory: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("not a directory: %s", path)
	}

	return afero.NewBasePathFs(base, path), nil
}
