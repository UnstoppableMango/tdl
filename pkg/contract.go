package tdl

import (
	"context"
	"fmt"
	"io"
	"io/fs"

	"github.com/spf13/afero"
	"github.com/unmango/go/iter"
	tdlv1alpha1 "github.com/unstoppablemango/tdl/pkg/unmango/dev/tdl/v1alpha1"
)

type Sink interface {
	WriteUnit(string, io.Reader) error
}

type Pipeline[T, V any] interface {
	Execute(context.Context, T, V) error
}

type SinkGenerator interface {
	Pipeline[*tdlv1alpha1.Spec, Sink]
}

type Generator interface {
	fmt.Stringer
	Execute(context.Context, *tdlv1alpha1.Spec) (afero.Fs, error)
}

type Plugin interface {
	fmt.Stringer
	SinkGenerator(Target) (SinkGenerator, error)
	Generator(context.Context, Target) (Generator, error)
}

type Target interface {
	fmt.Stringer
	Choose([]SinkGenerator) (SinkGenerator, error)
	Generator(iter.Seq[Plugin]) (Generator, error)
}

type MediaType string

// String implements fmt.Stringer.
func (m MediaType) String() string {
	return string(m)
}

type Input interface {
	io.Reader
	MediaType() MediaType
}

type Output interface {
	Write(afero.Fs) error
}

type ParseResult struct {
	Inputs []Input
	Output Output
}

type Stdin interface {
	io.Reader
	Stat() (fs.FileInfo, error)
}

type OS interface {
	Stdin() Stdin
	Stdout() io.Writer
	Stderr() io.Writer
	Fs() afero.Fs
}
