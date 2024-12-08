package progress

import (
	"github.com/unmango/go/rx"
	"github.com/unmango/go/rx/subject"
)

type Observable interface {
	rx.Observable[Event]
}

type Subject struct {
	rx.Subject[Event]
	State
}

type TotalObservable interface {
	rx.Observable[float64]
}

type MessageObservable interface {
	rx.Observable[string]
}

// OnComplete implements rx.Observer.
func (fn HandlerFunc) OnComplete() {}

// OnError implements rx.Observer.
func (fn HandlerFunc) OnError(err error) {
	fn(nil, err)
}

// OnNext implements rx.Observer.
func (fn HandlerFunc) OnNext(e Event) {
	fn(&e, nil)
}

func (fn HandlerFunc) SubscribeTo(obs Observable) rx.Subscription {
	return obs.Subscribe(fn)
}

// OnComplete implements rx.Observer.
func (TotalFunc) OnComplete() {}

// OnError implements rx.Observer.
func (fn TotalFunc) OnError(err error) {
	fn(0, err)
}

// OnNext implements rx.Observer.
func (r TotalFunc) OnNext(e Event) {
	r(e.Percent(), nil)
}

func (fn TotalFunc) SubscribeTo(obs Observable) rx.Subscription {
	return obs.Subscribe(fn)
}

// OnComplete implements rx.Observer.
func (fn MessageFunc) OnComplete() {}

// OnError implements rx.Observer.
func (fn MessageFunc) OnError(err error) {
	fn("", err)
}

// OnNext implements rx.Observer.
func (fn MessageFunc) OnNext(e Event) {
	fn(e.Message, nil)
}

func (fn MessageFunc) SubscribeTo(obs Observable) rx.Subscription {
	return obs.Subscribe(fn)
}

func NewSubject(total int) *Subject {
	return &Subject{
		Subject: subject.New[Event](),
		State:   &state{total: total},
	}
}

func Subscribe(obs Observable, handler HandlerFunc) rx.Subscription {
	return handler.SubscribeTo(obs)
}

func SubscribeTotal(obs Observable, handler TotalFunc) rx.Subscription {
	return handler.SubscribeTo(obs)
}

func SubscribeMessage(obs Observable, handler MessageFunc) rx.Subscription {
	return handler.SubscribeTo(obs)
}
