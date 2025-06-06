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

func (e Event) ProgressMsg() ProgressMsg {
	return ProgressMsg(e.Percent())
}

type (
	HandlerFunc func(*Event, error)
	TotalFunc   func(float64, error)
	MessageFunc func(string, error)
)

type Func interface {
	~func(*Event, error) | ~func(float64, error) | ~func(string, error)
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

func ChannelHandler(stream chan<- *Event, errs chan<- error) HandlerFunc {
	return func(e *Event, err error) {
		if err != nil {
			errs <- err
		} else {
			stream <- e
		}
	}
}
