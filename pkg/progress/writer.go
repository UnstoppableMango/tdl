package progress

import (
	"io"
)

type Writer interface {
	Observable
	io.Writer
}

type writer struct {
	Subject
	writer io.Writer
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
		e := w.Advance(n)
		w.OnNext(e.event())
	}

	return
}

func NewWriter(w io.Writer, total int) Writer {
	return &writer{
		Subject: NewSubject(total),
		writer:  w,
	}
}
