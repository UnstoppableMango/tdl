package testing

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/afero"
	"github.com/unstoppablemango/tdl/pkg/plugin"
)

type CacheForT struct {
	t   *testing.T
	dir string
}

// Cache implements plugin.Cacher.
func (c *CacheForT) Cache(bin string, data []byte) error {
	binary := filepath.Join(c.Dir(), bin)

	return os.WriteFile(binary, data, os.ModePerm)
}

func (c *CacheForT) Dir() string {
	if c.dir == "" {
		c.dir = c.t.TempDir()
	}

	return c.dir
}

type Cache struct {
	fs  afero.Fs
	dir string
}

// Cache implements plugin.Cacher.
func (c *Cache) Cache(bin string, data []byte) error {
	path, err := c.Dir()
	if err != nil {
		return err
	}

	binary := filepath.Join(path, bin)
	return afero.WriteFile(c.fs, binary, data, os.ModePerm)
}

func (c *Cache) Dir() (string, error) {
	if err := c.ensure(); err != nil {
		return "", nil
	}

	return c.dir, nil
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

var _ plugin.Cacher = &CacheForT{}
var _ plugin.Cacher = &Cache{}

func NewCacheForT(t *testing.T) *CacheForT {
	return &CacheForT{t: t}
}

func NewCache(fsys afero.Fs) *Cache {
	if fsys == nil {
		fsys = afero.NewOsFs()
	}

	return &Cache{fs: fsys}
}
