package testing

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/afero"
	"github.com/unstoppablemango/tdl/pkg/plugin/cache"
)

type CacheForT struct {
	t   *testing.T
	dir string
}

// Reader implements cache.Cacher.
func (c *CacheForT) Reader(name string) (io.Reader, error) {
	return os.Open(filepath.Join(c.Dir(), name))
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

// Reader implements cache.Cacher.
func (c *Cache) Reader(name string) (io.Reader, error) {
	dir, err := c.Dir()
	if err != nil {
		return nil, fmt.Errorf("reader for %s: %w", name, err)
	}

	return c.fs.Open(filepath.Join(dir, name))
}

// Cache implements plugin.Cacher.
func (c *Cache) Cache(name string, data []byte) error {
	path, err := c.Dir()
	if err != nil {
		return err
	}

	binary := filepath.Join(path, name)
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

var _ cache.Cacher = &CacheForT{}
var _ cache.Cacher = &Cache{}

func NewCacheForT(t *testing.T) *CacheForT {
	return &CacheForT{t: t}
}

func NewCache(fsys afero.Fs) *Cache {
	if fsys == nil {
		fsys = afero.NewOsFs()
	}

	return &Cache{fs: fsys}
}
