package pipe

import (
	"fmt"
	"io"
)

func MapSource[T any](source Source, fn func(string, io.Reader) (T, error)) (map[string]T, error) {
	result := map[string]T{}
	for unit := range source.Units() {
		r, err := source.Reader(unit)
		if err != nil {
			return nil, fmt.Errorf("reader lookup: %w", err)
		}

		if mapped, err := fn(unit, r); err != nil {
			return nil, fmt.Errorf("applying map: %w", err)
		} else {
			result[unit] = mapped
		}
	}

	return result, nil
}
