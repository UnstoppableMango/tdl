package plugin

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/adrg/xdg"
)

type Config interface {
	Cache(string, []byte) error
}

type userConfig struct {
	root string
}

var XdgConfig = userConfig{xdg.ConfigHome}

func (c userConfig) Cache(bin string, data []byte) error {
	if err := c.ensure(); err != nil {
		return fmt.Errorf("caching binary: %w", err)
	}

	return os.WriteFile(
		c.name(bin),
		data,
		os.ModePerm,
	)
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
