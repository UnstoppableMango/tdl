package cache

import (
	"errors"
	"io"
	"log/slog"
	"os"
	"path"
)

type fsCache struct {
	path string
	log  *slog.Logger
}

func NewFsCache(path string, logger *slog.Logger) Cache {
	return &fsCache{path: path, log: logger}
}

// Path implements Cache.
func (c *fsCache) Path(rel ...string) string {
	segments := append([]string{c.path}, rel...)
	return path.Join(segments...)
}

// Add implements Cache.
func (c *fsCache) Add(name string, reader io.Reader) error {
	if err := c.ensure(); err != nil {
		return err
	}

	itemPath := c.itemPath(name)
	c.log.Debug("creating file", "path", itemPath)
	file, err := os.Create(itemPath)
	if err != nil {
		return err
	}

	c.log.Debug("writing cache item", "path", itemPath)
	written, err := io.Copy(file, reader)
	if err != nil {
		return err
	}

	c.log.Debug("wrote cache item", "written", written)
	return nil
}

// Get implements Cache.
func (c *fsCache) Get(name string) (io.ReadCloser, error) {
	if err := c.ensure(); err != nil {
		return nil, err
	}

	itemPath := c.itemPath(name)
	c.log.Debug("opening file", "path", itemPath)
	file, err := os.Open(itemPath)
	if err != nil {
		return nil, err
	}

	c.log.Debug("opened cache item", "path", itemPath)
	return file, nil
}

// Remove implements Cache.
func (c *fsCache) Remove(name string) error {
	if err := c.ensure(); err != nil {
		return err
	}

	itemPath := c.itemPath(name)
	c.log.Debug("removing cache item", "path", itemPath)
	return os.Remove(itemPath)
}

func (c *fsCache) ensure() error {
	c.log.Debug("ensuring cache directory")
	err := os.Mkdir(c.path, 0750)
	if errors.Is(err, os.ErrExist) {
		return nil
	}

	return err
}

func (c *fsCache) itemPath(name string) string {
	return path.Join(c.path, name)
}

var _ Cache = &fsCache{}
