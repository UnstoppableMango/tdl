package testing

import (
	"testing"

	g "github.com/onsi/gomega"
	"github.com/spf13/afero"
	"github.com/unstoppablemango/tdl/pkg/plugin/cache"
)

type Cache struct{ cache.Directory }

func (c *Cache) Dir() string {
	return c.JoinPath()
}

var _ cache.Directory = &Cache{}

func NewCacheForT(t *testing.T) *Cache {
	return &Cache{
		cache.AtDirectory(t.TempDir()),
	}
}

// This needs a re-work. I want something that can be used
// easily in a test setup like `cache := testing.NewCache()`
// but there's the pesky little details of IO and the fact
// that it can error. With T this isn't so bad because we
// can t.Fail(). The current impl attempts similar behaviour
// but I don't like the "gotcha" that it must be run in a
// Ginkgo leaf node.

func NewCache(fsys afero.Fs) *Cache {
	if fsys == nil {
		fsys = afero.NewMemMapFs()
	}

	path, err := afero.TempDir(fsys, "", "")
	g.Expect(err).NotTo(g.HaveOccurred())

	return &Cache{cache.NewDirectory(fsys, path)}
}
