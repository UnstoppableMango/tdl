package output

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/afero"
	tdl "github.com/unstoppablemango/tdl/pkg"
	"github.com/unstoppablemango/tdl/pkg/sink"
)

func Fs(fsys afero.Fs, path string) (tdl.Sink, error) {
	stat, err := fsys.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		return fsFile(fsys, path)
	}
	if err != nil {
		return nil, err
	}
	if stat.IsDir() {
		return sink.NewFs(fsys, path), nil
	}

	return fsFile(fsys, path)
}

func ParseArgs(fsys afero.Fs, args []string) (tdl.Sink, error) {
	switch len(args) {
	case 0:
		fallthrough
	case 1:
		return sink.WriteTo(os.Stdout), nil
	case 2:
		return Fs(fsys, args[1])
	default:
		return nil, fmt.Errorf("unsupported args: %#v", args)
	}
}

func fsFile(fsys afero.Fs, path string) (tdl.Sink, error) {
	file, err := fsys.Create(path)
	if err != nil {
		return nil, err
	}

	return sink.WriteTo(file), nil
}
