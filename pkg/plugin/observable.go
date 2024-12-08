package plugin

import (
	"github.com/unmango/go/rx"
	tdl "github.com/unstoppablemango/tdl/pkg"
	"github.com/unstoppablemango/tdl/pkg/progress"
)

type Observable interface {
	tdl.Plugin
	progress.Observable
}

type noopObservable struct{ tdl.Plugin }

// Subscribe implements Observable.
func (n noopObservable) Subscribe(rx.Observer[progress.Event]) rx.Subscription {
	return func() {}
}

func Observe(plugin tdl.Plugin) Observable {
	if obs, ok := TryObserve(plugin); ok {
		return obs
	} else {
		return noopObservable{plugin}
	}
}

func TryObserve(plugin tdl.Plugin) (Observable, bool) {
	if obs, ok := plugin.(Observable); ok {
		return obs, true
	} else {
		return nil, false
	}
}

func TrySubscribe[F progress.Func](plugin tdl.Plugin, handler F) (rx.Subscription, bool) {
	if obs, ok := TryObserve(plugin); ok {
		return obs.Subscribe(progress.Lift(handler)), true
	} else {
		return nil, false
	}
}

func Subscribe[F progress.Func](plugin tdl.Plugin, handler F) rx.Subscription {
	return Observe(plugin).Subscribe(progress.Lift(handler))
}
