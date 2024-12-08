package progress

type TotalEvent struct {
	Current int
	Total   int
}

func (e TotalEvent) Percent() float64 {
	return float64(e.Current) / float64(e.Total)
}

func (e TotalEvent) event() Event {
	return Event{TotalEvent: e}
}

type MessageEvent struct {
	Message string
}

type Event struct {
	MessageEvent
	TotalEvent
}

type (
	HandlerFunc func(*Event, error)
	TotalFunc   func(float64, error)
	MessageFunc func(string, error)
)

type Func interface {
	HandlerFunc | TotalFunc | MessageFunc
}

func (fn TotalFunc) Handler() HandlerFunc {
	return func(e *Event, err error) {
		fn(e.Percent(), err)
	}
}

func (fn MessageFunc) Handler() HandlerFunc {
	return func(e *Event, err error) {
		fn(e.Message, err)
	}
}

func Lift[F Func](fn F) HandlerFunc {
	switch fn := any(fn).(type) {
	case HandlerFunc:
		return fn
	case TotalFunc:
		return fn.Handler()
	case MessageFunc:
		return fn.Handler()
	default:
		panic("invalid handler function")
	}
}
