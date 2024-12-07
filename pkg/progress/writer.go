package progress

import (
	"io"

	"github.com/unmango/go/rx"
	"github.com/unmango/go/rx/subject"
)

type Writer interface {
	Observable
	io.Writer
}

type writer struct {
	rx.Subject[Event]
	writer  io.Writer
	current int
	total   int
}

func (w *writer) Close() (err error) {
	c, ok := w.writer.(io.WriteCloser)
	if !ok {
		return
	}

	if err = c.Close(); err != nil {
		w.OnError(err)
	} else {
		w.OnComplete()
	}

	return
}

// Write implements io.Writer.
func (w *writer) Write(p []byte) (n int, err error) {
	n, err = w.writer.Write(p)
	if err != nil {
		w.OnError(err)
	} else {
		w.current += n
		p := float64(w.current) / float64(w.total)
		w.OnNext(Event{p})
	}

	return
}

func NewWriter(w io.Writer, total int) Writer {
	return &writer{
		Subject: subject.New[Event](),
		writer:  w,
		total:   total,
	}
}
