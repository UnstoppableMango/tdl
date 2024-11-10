package input

import (
	"github.com/spf13/afero"
	tdl "github.com/unstoppablemango/tdl/pkg"
	"github.com/unstoppablemango/tdl/pkg/mediatype"
)

type file struct {
	afero.File
	media tdl.MediaType
}

func (f *file) MediaType() tdl.MediaType {
	return f.media
}

func Open(fsys afero.Fs, path string) (tdl.Input, error) {
	input, err := fsys.Open(path)
	if err != nil {
		return nil, err
	}

	media, err := mediatype.Guess(path)
	if err != nil {
		return nil, err
	}

	return &file{input, media}, nil
}
