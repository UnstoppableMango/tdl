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

type Pipeline[T, V any] interface {
	Execute(context.Context, T) (V, error)
}

type Generator interface {
	fmt.Stringer
	Execute(context.Context, *tdlv1alpha1.Spec) (afero.Fs, error)
}

type Tool interface {
	fmt.Stringer
	Execute(context.Context, afero.Fs) (afero.Fs, error)
}

type Parser interface {
	fmt.Stringer
	Execute(context.Context, afero.Fs) (*tdlv1alpha1.Spec, error)
}

type GeneratorPlugin interface {
	Plugin
	Generator(context.Context, Meta) (Generator, error)
}

type ToolPlugin interface {
	Plugin
	Tool(context.Context, Meta) (Tool, error)
}

type Plugin interface {
	fmt.Stringer
	Meta() Meta
	Supports(Target) bool
}

type Meta interface {
	Values() iter.Seq2[string, string]
	Value(string) (string, bool)
}

type Target interface {
	fmt.Stringer
	Meta() Meta
	Choose(iter.Seq[Plugin]) (Plugin, error)
}

type MediaType string

// String implements fmt.Stringer.
func (m MediaType) String() string {
	return string(m)
}

type Input interface {
	fmt.Stringer
	io.Reader
	MediaType() MediaType
}

type Output interface {
	fmt.Stringer
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
