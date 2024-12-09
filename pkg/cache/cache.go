package cache

import (
	"fmt"
	"io"

	tdl "github.com/unstoppablemango/tdl/pkg"
)

type CreateFunc func() (io.ReadCloser, error)

func GetOrCreate(cache tdl.Cache, id string, create CreateFunc) (*tdl.CacheItem, error) {
	if item, err := cache.Get(id); err == nil {
		return item, nil
	}

	writer, err := cache.Writer(id)
	if err != nil {
		return nil, fmt.Errorf("opening cache: %w", err)
	}

	reader, err := create()
	if err != nil {
		return nil, fmt.Errorf("creating cache item: %w", err)
	}

	return &tdl.CacheItem{
		Name: id,
		ReadCloser: io.NopCloser(
			io.TeeReader(reader, writer),
		),
	}, nil
}
