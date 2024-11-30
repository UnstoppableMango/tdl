package internal

import (
	"context"
	"io"
	"os"

	"github.com/spf13/afero"
	tdl "github.com/unstoppablemango/tdl/pkg"
)

type key string

var (
	osKey key = "os"
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

func GetOs(ctx context.Context) tdl.OS {
	if v := ctx.Value(osKey); v != nil {
		return v.(tdl.OS)
	} else {
		return RealOs()
	}
}

func WithOs(parent context.Context, os tdl.OS) context.Context {
	return context.WithValue(parent, osKey, os)
}
