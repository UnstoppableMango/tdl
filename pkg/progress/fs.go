package progress

import (
	"github.com/spf13/afero"
	"github.com/unmango/go/rx"
)

type File interface {
	afero.File
	Reader
}

type file struct {
	afero.File
	r Reader
}

// Read implements File.
func (f *file) Read(p []byte) (n int, err error) {
	return f.r.Read(p)
}

// Subscribe implements File.
func (f *file) Subscribe(obs rx.Observer[Event]) rx.Subscription {
	return f.r.Subscribe(obs)
}

func Open(fs afero.Fs, name string) (File, error) {
	f, err := fs.Open(name)
	if err != nil {
		return nil, err
	}

	info, err := f.Stat()
	if err != nil {
		return nil, err
	}

	reader := NewReader(f, int(info.Size()))
	return &file{File: f, r: reader}, nil
}
