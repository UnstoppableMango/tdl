package io

import (
	"bytes"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"io"

	tdl "github.com/unstoppablemango/tdl/pkg"
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

type SinkWriter struct {
	sink tdl.Sink
}

// Write implements io.Writer.
func (s *SinkWriter) Write(p []byte) (n int, err error) {
	h := sha1.New()
	n, err = h.Write(p)
	if err != nil {
		return 0, fmt.Errorf("hashing unit: %w", err)
	}

	hash := base64.StdEncoding.EncodeToString(h.Sum(nil))
	if err = s.sink.WriteUnit(hash, bytes.NewReader(p)); err != nil {
		return 0, fmt.Errorf("writing to inner sink: %w", err)
	}

	return n, nil
}

func NewSink(writer io.Writer) tdl.Sink {
	return &Sink{Writer: writer}
}

func NewSinkWriter(sink tdl.Sink) io.Writer {
	return &SinkWriter{sink}
}
