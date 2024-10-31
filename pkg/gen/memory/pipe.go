package memory

import (
	"bytes"
	"fmt"
	"io"
	"iter"
	"maps"

	"github.com/unstoppablemango/tdl/pkg/tdl"
)

type PorcelainPipe struct {
	units map[string]io.Reader
}

// Reader implements tdl.Pipe.
func (p *PorcelainPipe) Reader(unit string) (io.Reader, error) {
	if r, ok := p.units[unit]; !ok {
		return nil, fmt.Errorf("no reader for unit: %s", unit)
	} else {
		return r, nil
	}
}

// Units implements tdl.Pipe.
func (p *PorcelainPipe) Units() iter.Seq[string] {
	return maps.Keys(p.units)
}

// WriteUnit implements tdl.Pipe.
func (p *PorcelainPipe) WriteUnit(unit string, reader io.Reader) error {
	p.units[unit] = reader
	return nil
}

type BufferedPipe struct {
	units map[string]*bytes.Buffer
}

// Reader implements tdl.Pipe.
func (p *BufferedPipe) Reader(unit string) (io.Reader, error) {
	if r, ok := p.units[unit]; !ok {
		return nil, fmt.Errorf("no reader for unit: %s", unit)
	} else {
		return r, nil
	}
}

// Units implements tdl.Pipe.
func (p *BufferedPipe) Units() iter.Seq[string] {
	return maps.Keys(p.units)
}

// WriteUnit implements tdl.Pipe.
func (p *BufferedPipe) WriteUnit(unit string, reader io.Reader) error {
	if data, err := io.ReadAll(reader); err != nil {
		return fmt.Errorf("reading unit: %w", err)
	} else {
		p.units[unit] = bytes.NewBuffer(data)
	}

	return nil
}

func NewPipe() tdl.Pipe {
	return &PorcelainPipe{
		units: map[string]io.Reader{},
	}
}
