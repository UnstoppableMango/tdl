package cache

import (
	"fmt"
	"io"
	"os"

	"github.com/charmbracelet/log"
	"github.com/spf13/afero"
	tdl "github.com/unstoppablemango/tdl/pkg"
)

type Fs struct{ afero.Fs }

// Get implements tdl.Cache.
func (f *Fs) Get(key string) (*tdl.CacheItem, error) {
	file, err := f.Open(key)
	if err != nil {
		log.Debug("opening cache file", "err", err)
		return nil, keyDoesNotExist(key)
	}

	return &tdl.CacheItem{
		ReadCloser: file,
		Name:       key,
	}, nil
}

// Writer implements tdl.Cache.
func (f *Fs) Writer(key string) (io.WriteCloser, error) {
	file, err := f.Open(key)
	if os.IsNotExist(err) {
		file, err = f.Create(key)
	}
	if err != nil {
		return nil, fmt.Errorf("opening cached file: %w", err)
	}

	return file, nil
}

var _ tdl.Cache = &Fs{}

func NewFs(fs afero.Fs) *Fs {
	return &Fs{fs}
}

func NewFsAt(fs afero.Fs, path string) (*Fs, error) {
	if err := ensure(fs, path); err != nil {
		return nil, fmt.Errorf("ensuring cache exists: %w", err)
	}

	return NewFs(afero.NewBasePathFs(fs, path)), nil
}

func NewMemFs() *Fs {
	return NewFs(afero.NewMemMapFs())
}

func NewTmpFs() (*Fs, error) {
	fs := afero.NewOsFs()
	tmp, err := afero.TempDir(fs, "", "")
	if err != nil {
		return nil, fmt.Errorf("creating tmp dir: %w", err)
	}

	// TODO: Maybe implement io.Closer to delete the tmp dir
	return NewFsAt(fs, tmp)
}

func ensure(fs afero.Fs, path string) error {
	stat, err := fs.Stat(path)
	if err == nil {
		if stat.IsDir() {
			return nil
		} else {
			return fmt.Errorf("cache must be a directory: %s", path)
		}
	}
	if os.IsNotExist(err) {
		log.Debugf("creating cache directory: %s", path)
		err = fs.MkdirAll(path, os.ModePerm)
	}

	return err
}
