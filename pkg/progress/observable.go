package progress

import (
	"errors"

	"github.com/unmango/go/rx"
)

var ErrCompleted = errors.New("done")

// OnComplete implements rx.Observer.
func (r ReportFunc) OnComplete() {
	r(0, ErrCompleted)
}

// OnError implements rx.Observer.
func (r ReportFunc) OnError(err error) {
	r(0, err)
}

// OnNext implements rx.Observer.
func (r ReportFunc) OnNext(e Event) {
	r(e.N, nil)
}

type Observable interface {
	rx.Observable[Event]
}

func Subscribe(o Observable, report ReportFunc) rx.Subscription {
	return o.Subscribe(report)
}
