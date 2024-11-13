package progress

import (
	"github.com/unmango/go/rx"
)

// OnComplete implements rx.Observer.
func (r ReportFunc) OnComplete() {}

// OnError implements rx.Observer.
func (r ReportFunc) OnError(err error) {
	r(0, err)
}

// OnNext implements rx.Observer.
func (r ReportFunc) OnNext(e Event) {
	r(e.Percent, nil)
}

type Observable interface {
	rx.Observable[Event]
}

func Subscribe(o Observable, report ReportFunc) rx.Subscription {
	return o.Subscribe(report)
}
