package sink

import (
	"fmt"
	"io"
	"iter"
	"slices"

	"github.com/spf13/afero"
	tdl "github.com/unstoppablemango/tdl/pkg"
)

type Fs interface {
	Reader
	tdl.Sink
}

type fsSink struct {
	afero.Fs
	files []string
}

// Reader implements Fs.
func (f *fsSink) Reader(path string) (io.Reader, error) {
	if !slices.Contains(f.files, path) {
		return nil, fmt.Errorf("unknown unit: %s", path)
	}

	return f.Open(path)
}

// Units implements Fs.
func (f *fsSink) Units() iter.Seq[string] {
	return slices.Values(f.files)
}

// WriteUnit implements tdl.Sink.
func (f *fsSink) WriteUnit(path string, reader io.Reader) error {
	err := afero.WriteReader(f, path, reader)
	if err != nil {
		return fmt.Errorf("writing unit: %w", err)
	}

	f.files = append(f.files, path)
	return nil
}

func NewFs(fsys afero.Fs, path string) Fs {
	return &fsSink{
		afero.NewBasePathFs(fsys, path),
		[]string{},
	}
}
