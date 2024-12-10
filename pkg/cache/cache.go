package cache

import (
	"errors"
	"fmt"
	"io"

	"github.com/charmbracelet/log"
	tdl "github.com/unstoppablemango/tdl/pkg"
)

var ErrNotExist = errors.New("cache key does not exist")

func IsNotExist(err error) bool {
	return errors.Is(err, ErrNotExist)
}

type CreateFunc func() (io.ReadCloser, error)

func Exists(cache tdl.Cache, key string) bool {
	_, err := cache.Get(key)
	return err == nil
}

func GetOrCreate(cache tdl.Cache, key string, create CreateFunc) (*tdl.CacheItem, error) {
	if item, err := cache.Get(key); err == nil {
		return item, nil
	}

	writer, err := cache.Writer(key)
	if err != nil {
		return nil, fmt.Errorf("opening cache: %w", err)
	}

	reader, err := create()
	if err != nil {
		return nil, fmt.Errorf("creating cache item: %w", err)
	}

	return &tdl.CacheItem{
		Name:       key,
		ReadCloser: newTeeCloser(reader, writer),
	}, nil
}

func WriteAll(cache tdl.Cache, key string, reader io.Reader) error {
	writer, err := cache.Writer(key)
	if err != nil {
		return fmt.Errorf("opening cache: %w", err)
	}

	written, err := io.Copy(writer, reader)
	if err != nil {
		return fmt.Errorf("writing cache item: %w", err)
	}

	log.Debugf("wrote %d bytes to cache key %s", written, key)
	return nil
}

func WriteString(cache tdl.Cache, key string, data string) error {
	writer, err := cache.Writer(key)
	if err != nil {
		return fmt.Errorf("opening cache: %w", err)
	}

	written, err := io.WriteString(writer, data)
	if err != nil {
		return fmt.Errorf("writing cache item: %w", err)
	}

	log.Debugf("wrote %d bytes to cache key %s", written, key)
	return nil
}

func keyDoesNotExist(key string) error {
	return fmt.Errorf("%w: %s", ErrNotExist, key)
}
