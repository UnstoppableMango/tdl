package cache

import (
	"io"
)

type Cache interface {
	Add(string, io.Reader) error
	Get(string) (io.ReadCloser, error)
	Path(string) (string, error)
	Remove(string) error
}

func GetOrAdd(cache Cache, name string, reader io.Reader) (io.ReadCloser, error) {
	if item, err := cache.Get(name); err == nil {
		return item, nil
	}

	if err := cache.Add(name, reader); err != nil {
		return nil, err
	}

	return cache.Get(name)
}
