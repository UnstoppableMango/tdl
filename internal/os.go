package internal

import (
	"io"
	"os"

	"github.com/spf13/afero"
	tdl "github.com/unstoppablemango/tdl/pkg"
)

type realos struct {
	fs afero.Fs
}

// Fs implements tdl.OS.
func (i realos) Fs() afero.Fs {
	return i.fs
}

// Stderr implements tdl.OS.
func (realos) Stderr() io.Writer {
	return os.Stderr
}

// Stdin implements tdl.OS.
func (realos) Stdin() tdl.Stdin {
	return os.Stdin
}

// Stdout implements tdl.OS.
func (realos) Stdout() io.Writer {
	return os.Stdout
}

func RealOs() tdl.OS {
	return realos{afero.NewOsFs()}
}
