package progress

import (
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

	r.Subject.OnComplete()
	return
}

// Read implements io.ReadCloser.
func (r *reader) Read(p []byte) (n int, err error) {
	n, err = r.reader.Read(p)
	if err != nil {
		r.Subject.OnError(err)
	} else {
		r.Subject.OnNext(Event{n})
	}

	return
}

func NewReader(r io.Reader) Reader {
	return &reader{
		Subject: subject.New[Event](),
		reader:  r,
	}
}
