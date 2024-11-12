package cache

import (
	"errors"
	"io"

	"github.com/unmango/go/rx"
	"github.com/unmango/go/rx/subject"
	"github.com/unstoppablemango/tdl/pkg/progress"
)

var (
	ErrCompleted = errors.New("cache observable has completed")
)

type CacheEvent struct {
	Name string
	Len  int
}

type Observable interface {
	rx.Observable[CacheEvent]
	Cacher
}

type ObserveFunc func(string, int, error)

// OnComplete implements rx.Observer.
func (o ObserveFunc) OnComplete() {
	o("", 0, ErrCompleted)
}

// OnError implements rx.Observer.
func (o ObserveFunc) OnError(err error) {
	o("", 0, err)
}

// OnNext implements rx.Observer.
func (o ObserveFunc) OnNext(e CacheEvent) {
	o(e.Name, e.Len, nil)
}

type observable struct {
	rx.Subject[CacheEvent]
	cache Cacher
}

// Reader implements Observable.
func (o *observable) Reader(name string) (io.Reader, error) {
	return o.cache.Reader(name)
}

// Write implements Observable.
func (o *observable) Write(name string, data []byte) error {
	err := o.cache.Write(name, data)
	if err != nil {
		o.Subject.OnError(err)
	} else {
		o.Subject.OnNext(CacheEvent{name, len(data)})
		o.Subject.OnComplete()
	}

	return err
}

func Observe(cache Cacher) Observable {
	return &observable{
		Subject: subject.New[CacheEvent](),
		cache:   cache,
	}
}

func Report(cache Cacher, report progress.ReportFunc) rx.Subscription {
	return Subscribe(Observe(cache),
		func(_ string, i int, err error) {
			report(i, err)
		},
	)
}

func Subscribe(obs Observable, report ObserveFunc) rx.Subscription {
	return obs.Subscribe(report)
}
