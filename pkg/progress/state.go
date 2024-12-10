package progress

type State interface {
	Advance(int) TotalEvent
}

type state struct {
	current, total int
}

func (s *state) Advance(n int) TotalEvent {
	s.current += n

	return TotalEvent{
		Current: s.current,
		Total:   s.total,
	}
}
