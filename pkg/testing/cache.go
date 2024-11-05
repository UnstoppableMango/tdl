package testing

import (
	"testing"

	"github.com/spf13/afero"
	"github.com/unstoppablemango/tdl/pkg/plugin/cache"
)

type Cache struct {
	cache.Directory
}

func (c *Cache) ensure() error {
	if c.dir != "" {
		return nil
	}

	path, err := afero.TempDir(c.fs, "", "")
	if err != nil {
		return err
	}

	c.dir = path
	return nil
}

var _ cache.Cacher = &Cache{}

func NewCacheForT(t *testing.T) *Cache {
	return &Cache{cache.AtDirectory(t.TempDir())}
}

func NewCache(fsys afero.Fs) *Cache {
	cache := cache.AtDirectory(path)
	return &Cache{Directory: cache}
}
