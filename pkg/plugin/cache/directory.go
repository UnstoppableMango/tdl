package cache

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/charmbracelet/log"
	"github.com/spf13/afero"
)

type Directory interface {
	Cacher
	JoinPath(...string) string
	List() ([]string, error)
}

type directory struct {
	fs   afero.Fs
	path string
}

// JoinPath implements Directory.
func (d *directory) JoinPath(elem ...string) string {
	elem = append([]string{d.path}, elem...)
	return filepath.Join(elem...)
}

// Write implements Cacher.
func (d *directory) Write(name string, data []byte) error {
	if err := d.ensure(); err != nil {
		return fmt.Errorf("caching binary: %w", err)
	}

	return afero.WriteFile(d.fs,
		d.rel(name),
		data,
		os.ModePerm,
	)
}

// Reader implements Cacher.
func (d *directory) Reader(name string) (io.Reader, error) {
	return d.fs.Open(d.rel(name))
}

func (d *directory) List() ([]string, error) {
	keys := []string{}
	err := afero.Walk(d.fs, d.path,
		func(path string, info fs.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				return nil
			}

			rel, err := filepath.Rel(d.path, path)
			if err != nil {
				return err
			}

			keys = append(keys, rel)
			return nil
		},
	)
	if err != nil {
		return nil, err
	}

	return keys, nil
}

func (d directory) ensure() error {
	stat, err := d.fs.Stat(d.path)
	if errors.Is(err, os.ErrNotExist) {
		log.Debug("creating cache directory", "path", d.path)
		return d.fs.MkdirAll(d.path, os.ModePerm)
	}
	if err != nil {
		return err
	}
	if !stat.IsDir() {
		return fmt.Errorf("%s is not a directory", d.path)
	}

	return nil
}

func (d directory) rel(name string) string {
	return filepath.Join(d.path, name)
}

var _ Cacher = &directory{}

func AtDirectory(path string) Directory {
	return &directory{
		fs:   afero.NewOsFs(),
		path: path,
	}
}

func NewDirectory(fsys afero.Fs, path string) Directory {
	return &directory{fsys, path}
}
