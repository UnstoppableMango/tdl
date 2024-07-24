package cache

import (
	"errors"
	"io"
	"log/slog"
	"os"
	"path"
	"sync"
)

type fsCache struct {
	path  string
	locks map[string]*sync.RWMutex
	log   *slog.Logger
}

func NewFsCache(path string, logger *slog.Logger) Cache {
	return &fsCache{path: path, log: logger, locks: map[string]*sync.RWMutex{}}
}

// Path implements Cache.
func (c *fsCache) Path(name string) (string, error) {
	p := path.Join(c.path, name)
	c.lock(name).RLock()
	_, err := os.Stat(p)
	c.lock(name).RUnlock()

	return p, err
}

// Add implements Cache.
func (c *fsCache) Add(name string, reader io.Reader) error {
	if err := c.ensure(); err != nil {
		return err
	}

	itemPath := c.itemPath(name)
	c.log.Debug("creating file", "path", itemPath)
	c.lock(name).Lock()
	file, err := os.Create(itemPath)
	if err != nil {
		return err
	}

	c.log.Debug("writing cache item", "path", itemPath)
	written, err := io.Copy(file, reader)
	c.lock(name).Unlock()
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
	c.lock(name).RLock()
	file, err := os.Open(itemPath)
	c.lock(name).RUnlock()
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
	c.lock(name).Lock()
	err := os.Remove(itemPath)
	c.lock(name).Unlock()

	return err
}

func (c *fsCache) ensure() error {
	c.log.Debug("ensuring cache directory")
	err := os.MkdirAll(c.path, os.ModePerm)
	if errors.Is(err, os.ErrExist) {
		return nil
	}

	return err
}

func (c *fsCache) itemPath(name string) string {
	return path.Join(c.path, name)
}

func (c *fsCache) lock(name string) *sync.RWMutex {
	lock, ok := c.locks[name]
	if !ok {
		lock = &sync.RWMutex{}
		c.locks[name] = lock
	}

	return lock
}
