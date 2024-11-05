package testing

import (
	"io"
	"testing"

	"github.com/spf13/afero"
	"github.com/unstoppablemango/tdl/pkg/plugin/cache"
)

type Cache struct {
	fs   afero.Fs
	path string

	base cache.Directory
}

// List implements cache.Directory.
func (c *Cache) List() ([]string, error) {
	panic("unimplemented")
}

// Reader implements cache.Cacher.
func (c *Cache) Reader(string) (io.Reader, error) {
	panic("unimplemented")
}

// Write implements cache.Cacher.
func (c *Cache) Write(string, []byte) error {
	panic("unimplemented")
}

func (c *Cache) Dir() string {
	return c.path
}

var _ cache.Directory = &Cache{}

func NewCacheForT(t *testing.T) *Cache {
	return &Cache{
		fs:   afero.NewOsFs(),
		path: t.TempDir(),
	}
}

func NewCache(fsys afero.Fs) *Cache {
	if fsys == nil {
		fsys = afero.NewMemMapFs()
	}

	return &Cache{fs: fsys}
}
