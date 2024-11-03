package sink

import (
	"io"
	"iter"

	"github.com/unstoppablemango/tdl/pkg/tdl"
)

type Reader interface {
	Units() iter.Seq[string]
	Reader(string) (io.Reader, error)
}

type Pipe interface {
	Reader
	tdl.Sink
}
