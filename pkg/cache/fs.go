package cache

import (
	"fmt"
	"io"

	"github.com/spf13/afero"
	tdl "github.com/unstoppablemango/tdl/pkg"
)

type Fs struct {
	afero.Fs
}

// Get implements tdl.Cache.
func (f *Fs) Get(key string) (*tdl.CacheItem, error) {
	file, err := f.Open(key)
	if err != nil {
		return nil, fmt.Errorf("opening cached file: %w", err)
	}

	return &tdl.CacheItem{
		ReadCloser: file,
		Name:       file.Name(),
	}, nil
}

// Writer implements tdl.Cache.
func (f *Fs) Writer(key string) (io.WriteCloser, error) {
	return f.Open(key)
}

var _ tdl.Cache = &Fs{}

func NewFs(fs afero.Fs) *Fs {
	return &Fs{fs}
}
