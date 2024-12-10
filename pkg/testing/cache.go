package testing

import (
	"io"

	"github.com/onsi/ginkgo/v2"
	"github.com/spf13/afero"
	tdl "github.com/unstoppablemango/tdl/pkg"
	"github.com/unstoppablemango/tdl/pkg/cache"
)

type Cache struct {
	GetFunc    func(string) (*tdl.CacheItem, error)
	WriterFunc func(string) (io.WriteCloser, error)
}

func (c *Cache) Get(key string) (*tdl.CacheItem, error) {
	if c.GetFunc == nil {
		panic("unimplemented")
	}

	return c.GetFunc(key)
}

func (c *Cache) Writer(key string) (io.WriteCloser, error) {
	if c.WriterFunc == nil {
		panic("unimplemented")
	}

	return c.WriterFunc(key)
}

func NewCacheFrom(cache tdl.Cache) *Cache {
	return &Cache{
		GetFunc:    cache.Get,
		WriterFunc: cache.Writer,
	}
}

func NewTmpCache(t ginkgo.GinkgoTInterface) (*Cache, string) {
	tmp := t.TempDir()
	fs := afero.NewBasePathFs(afero.NewOsFs(), tmp)

	return NewCacheFrom(cache.NewFs(fs)), tmp
}
