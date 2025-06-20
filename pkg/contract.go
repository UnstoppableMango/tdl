package tdl

import (
	"io"
)

type Parser[T any] interface {
	Parse(string) (T, error)
}

type MediaType string

// String implements fmt.Stringer.
func (m MediaType) String() string {
	return string(m)
}

type CacheItem struct {
	io.ReadCloser
	Name string
	Size int
}

type Cache interface {
	Get(string) (*CacheItem, error)
	Writer(string) (io.WriteCloser, error)
}
