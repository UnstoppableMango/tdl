package cache

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/adrg/xdg"
)

type Cacher interface {
	Cache(string, []byte) error
	Reader(string) (io.Reader, error)
}

type userConfig struct {
	root string
}

// Reader implements Cacher.
func (c userConfig) Reader(name string) (io.Reader, error) {
	return os.Open(c.name(name))
}

var XdgConfig = userConfig{xdg.ConfigHome}

func (c userConfig) Cache(name string, data []byte) error {
	if err := c.ensure(); err != nil {
		return fmt.Errorf("caching binary: %w", err)
	}

	return os.WriteFile(
		c.name(name),
		data,
		os.ModePerm,
	)
}

func All(cache Cacher, name string, reader io.Reader) error {
	data, err := io.ReadAll(reader)
	if err != nil {
		return err
	}

	return cache.Cache(name, data)
}

func (c userConfig) ensure() error {
	binDir := c.binDir()
	stat, err := os.Stat(binDir)
	if err != nil {
		return err
	}
	if stat.IsDir() {
		return nil
	}

	return os.MkdirAll(binDir, os.ModeDir)
}

func (c userConfig) binDir() string {
	return filepath.Join(c.root, "bin")
}

func (c userConfig) name(name string) string {
	return filepath.Join(c.binDir(), name)
}

var _ Cacher = &userConfig{}
