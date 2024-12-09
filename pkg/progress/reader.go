package progress

import (
	"errors"
	"io"
)

type Reader interface {
	Observable
	io.Reader
}

type reader struct {
	Subject
	reader io.Reader
}

// Close implements io.ReadCloser.
func (r *reader) Close() (err error) {
	if c, ok := r.reader.(io.ReadCloser); ok {
		err = c.Close()
	}
	if err != nil {
		r.Subject.OnError(err)
	}

	// TODO: Fix double OnComplete call
	r.OnComplete()
	return
}

// Read implements io.ReadCloser.
func (r *reader) Read(p []byte) (n int, err error) {
	n, err = r.reader.Read(p)
	if errors.Is(err, io.EOF) {
		r.Subject.OnComplete()
	} else if err != nil {
		r.Subject.OnError(err)
	} else {
		e := r.Advance(n)
		r.OnNext(e.event())
	}

	return
}

func NewReader(r io.Reader, total int) Reader {
	return &reader{
		Subject: NewSubject(total),
		reader:  r,
	}
}
