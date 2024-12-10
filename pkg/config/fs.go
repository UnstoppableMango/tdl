package config

import (
	"fmt"
	"io"

	"github.com/spf13/afero"
	"github.com/unstoppablemango/tdl/pkg/cache"
	"github.com/unstoppablemango/tdl/pkg/config/paths"
)

func BinExists(name string) bool {
	if bins, err := XdgBin(); err != nil {
		return false
	} else {
		return cache.Exists(bins, name)
	}
}

func XdgCache() (*cache.Fs, error) {
	return cache.NewFsAt(afero.NewOsFs(), paths.XdgCacheHome)
}

func XdgBin() (*cache.Fs, error) {
	return cache.NewFsAt(afero.NewOsFs(), paths.XdgBinHome)
}

func WriteBin(name string, reader io.Reader) error {
	if bins, err := XdgBin(); err != nil {
		return fmt.Errorf("opening bin directory: %w", err)
	} else {
		return cache.WriteAll(bins, name, reader)
	}
}
