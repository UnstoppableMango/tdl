package cache

import (
	"errors"
	"io"
)

type teeCloser struct {
	io.Reader
	closers []io.Closer
}

func (tee *teeCloser) Close() (err error) {
	for _, close := range tee.closers {
		err = errors.Join(err, close.Close())
	}

	return
}

func newTeeCloser(r io.Reader, w io.Writer) io.ReadCloser {
	closers := []io.Closer{}
	if closer, ok := r.(io.Closer); ok {
		closers = append(closers, closer)
	}
	if closer, ok := w.(io.Closer); ok {
		closers = append(closers, closer)
	}

	return &teeCloser{
		Reader:  io.TeeReader(r, w),
		closers: closers,
	}
}
