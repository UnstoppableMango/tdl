package io

import (
	"fmt"
	"io"

	"github.com/unstoppablemango/tdl/pkg/tdl"
)

// The io Sink discards unit names and writes all output
// to the provided io.Writer
type Sink struct {
	Writer io.Writer
}

// WriteUnit implements tdl.Sink.
func (s *Sink) WriteUnit(_ string, reader io.Reader) error {
	if _, err := io.Copy(s.Writer, reader); err != nil {
		return fmt.Errorf("copying unit: %w", err)
	}

	return nil
}

func NewSink(writer io.Writer) tdl.Sink {
	return &Sink{Writer: writer}
}
