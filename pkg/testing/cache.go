package testing

import (
	"io"
	"testing"

	"github.com/onsi/ginkgo/v2"
	g "github.com/onsi/gomega"
	"github.com/spf13/afero"
	tdl "github.com/unstoppablemango/tdl/pkg"
	"github.com/unstoppablemango/tdl/pkg/cache"
	pcache "github.com/unstoppablemango/tdl/pkg/plugin/cache"
)

type DirCache struct{ pcache.Directory }

func (c *DirCache) Dir() string {
	return c.JoinPath()
}

var _ pcache.Directory = &DirCache{}

func NewCacheForT(t *testing.T) *DirCache {
	return &DirCache{
		pcache.AtDirectory(t.TempDir()),
	}
}

// This needs a re-work. I want something that can be used
// easily in a test setup like `cache := testing.NewCache()`
// but there's the pesky little details of IO and the fact
// that it can error. With T this isn't so bad because we
// can t.Fail(). The current impl attempts similar behaviour
// but I don't like the "gotcha" that it must be run in a
// Ginkgo leaf node.

func NewCache(fsys afero.Fs) *DirCache {
	if fsys == nil {
		fsys = afero.NewMemMapFs()
	}

	path, err := afero.TempDir(fsys, "", "")
	g.Expect(err).NotTo(g.HaveOccurred())

	return &DirCache{pcache.NewDirectory(fsys, path)}
}

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
