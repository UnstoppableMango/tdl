package gen

import (
	"fmt"
	"io"

	"github.com/unstoppablemango/tdl/pkg/sink"
)

func MapSource[T any](source sink.Reader, fn func(string, io.Reader) (T, error)) (result map[string]T, err error) {
	readers, err := sink.Readers(source)
	if err != nil {
		return nil, err
	}

	result = make(map[string]T, len(readers))
	for unit, r := range readers {
		if mapped, err := fn(unit, r); err != nil {
			return nil, fmt.Errorf("applying map: %w", err)
		} else {
			result[unit] = mapped
		}
	}

	return
}
