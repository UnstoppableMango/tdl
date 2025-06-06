package run

import (
	"errors"
	"fmt"
	"io/fs"
	"os"

	"github.com/charmbracelet/log"
	"github.com/spf13/afero"
	tdl "github.com/unstoppablemango/tdl/pkg"
	"github.com/unstoppablemango/tdl/pkg/mediatype"
)

type file struct {
	afero.File
	media tdl.MediaType
}

// String implements tdl.Input.
func (f *file) String() string {
	return fmt.Sprintf("file: %s %s", f.Name(), f.media)
}

func (f *file) MediaType() tdl.MediaType {
	return f.media
}

func OpenFile(fsys afero.Fs, path string) (tdl.Input, error) {
	stat, err := fsys.Stat(path)
	if err != nil {
		return nil, err
	}
	if stat.IsDir() {
		return nil, fmt.Errorf("%s is a directory", path)
	}

	f, err := fsys.Open(path)
	if err != nil {
		return nil, err
	}

	media, err := mediatype.Guess(path)
	if err != nil {
		return nil, err
	}

	return &file{f, media}, nil
}

type fsOutput struct {
	dest afero.Fs
	path string
}

// String implements tdl.Output.
func (f *fsOutput) String() string {
	return fmt.Sprintf("fs: %s %s", f.dest.Name(), f.path)
}

func (f *fsOutput) Write(output afero.Fs) error {
	stat, err := f.dest.Stat(f.path)
	if errors.Is(err, os.ErrNotExist) {
		return f.writeFile(output)
	}
	if err != nil {
		return err
	}
	if stat.IsDir() {
		return copyFs(output, afero.NewBasePathFs(f.dest, f.path))
	}

	return f.writeFile(output)
}

func (f *fsOutput) writeFile(output afero.Fs) error {
	file, err := f.dest.Create(f.path)
	if err != nil {
		return err
	}

	log.Debugf("writing to file %s", file.Name())
	return WriterOutput(file).Write(output)
}

func FsOutput(dest afero.Fs, path string) tdl.Output {
	return &fsOutput{dest, path}
}

func copyFs(src, dest afero.Fs) error {
	log.Debugf("copying fs %s to fs %s", src.Name(), dest.Name())
	return afero.Walk(src, "",
		func(path string, info fs.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if path == "" {
				return nil // Skip root
			}
			if info.IsDir() {
				return dest.Mkdir(path, os.ModeDir)
			}

			if file, err := src.Open(path); err != nil {
				return nil
			} else {
				return afero.WriteReader(dest, path, file)
			}
		},
	)
}
