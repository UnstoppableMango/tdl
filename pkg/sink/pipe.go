package sink

import (
	"bytes"
	"fmt"
	"io"
	"iter"
	"maps"

	tdl "github.com/unstoppablemango/tdl/pkg"
)

type Memory interface {
	Reader
	tdl.Sink
}

type Porcelain struct {
	units map[string]io.Reader
}

// Reader implements tdl.Pipe.
func (p *Porcelain) Reader(unit string) (io.Reader, error) {
	if p.units == nil {
		return nil, notFoundErr(unit)
	}

	if r, ok := p.units[unit]; !ok {
		return nil, notFoundErr(unit)
	} else {
		return r, nil
	}
}

// Units implements tdl.Pipe.
func (p *Porcelain) Units() iter.Seq[string] {
	return maps.Keys(p.units)
}

// WriteUnit implements tdl.Pipe.
func (p *Porcelain) WriteUnit(unit string, reader io.Reader) error {
	if p.units == nil {
		p.units = make(map[string]io.Reader)
	}

	p.units[unit] = reader
	return nil
}

type Buffered struct {
	units map[string]*bytes.Buffer
}

// Reader implements tdl.Pipe.
func (p *Buffered) Reader(unit string) (io.Reader, error) {
	if p.units == nil {
		return nil, notFoundErr(unit)
	}

	if r, ok := p.units[unit]; !ok {
		return nil, fmt.Errorf("no reader for unit: %s", unit)
	} else {
		return r, nil
	}
}

// Units implements tdl.Pipe.
func (p *Buffered) Units() iter.Seq[string] {
	return maps.Keys(p.units)
}

// WriteUnit implements tdl.Pipe.
func (p *Buffered) WriteUnit(unit string, reader io.Reader) error {
	if p.units == nil {
		p.units = make(map[string]*bytes.Buffer)
	}

	if data, err := io.ReadAll(reader); err != nil {
		return fmt.Errorf("reading unit: %w", err)
	} else {
		p.units[unit] = bytes.NewBuffer(data)
	}

	return nil
}

func NewPipe() Memory {
	return &Porcelain{
		units: make(map[string]io.Reader),
	}
}

func notFoundErr(unit string) error {
	return fmt.Errorf("no reader for unit: %s", unit)
}