package plugin

import (
	"github.com/charmbracelet/log"
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
	log.Debugf("noop subscribe: %s", n.Plugin)
	return func() {}
}

func Observe(plugin tdl.Plugin) Observable {
	if obs, ok := TryObserve(plugin); ok {
		log.Debugf("observing plugin: %s", plugin)
		return obs
	} else {
		log.Debugf("not an observable plugin: %s", plugin)
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
		log.Debugf("subscribing to %s", plugin)
		return obs.Subscribe(progress.Lift(handler)), true
	} else {
		return nil, false
	}
}

func Subscribe[F progress.Func](plugin tdl.Plugin, handler F) rx.Subscription {
	return Observe(plugin).Subscribe(progress.Lift(handler))
}
