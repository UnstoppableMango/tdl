package sink

import "io"

// Readers creates a map of unit to [io.Reader]
func Readers(sink Reader) (readers map[string]io.Reader, err error) {
	for u := range sink.Units() {
		if r, err := sink.Reader(u); err != nil {
			return nil, err
		} else {
			readers[u] = r
		}
	}

	return
}
