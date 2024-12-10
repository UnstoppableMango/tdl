package config

import (
	"github.com/spf13/afero"
	"github.com/unstoppablemango/tdl/pkg/cache"
	"github.com/unstoppablemango/tdl/pkg/paths"
)

func XdgCache() (*cache.Fs, error) {
	return cache.NewFsAt(afero.NewOsFs(), paths.XdgCacheHome)
}
