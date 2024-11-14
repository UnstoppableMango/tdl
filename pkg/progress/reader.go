package progress

import (
	"errors"
	"io"

	"github.com/unmango/go/rx"
	"github.com/unmango/go/rx/subject"
)

type Reader interface {
	Observable
	io.Reader
}

type reader struct {
	rx.Subject[Event]
	reader  io.Reader
	current int
	total   int
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
	r.Subject.OnComplete()
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
		r.current += n
		p := float64(r.current) / float64(r.total)
		r.Subject.OnNext(Event{p})
	}

	return
}

func NewReader(r io.Reader, total int) Reader {
	return &reader{
		Subject: subject.New[Event](),
		reader:  r,
		total:   total,
	}
}
