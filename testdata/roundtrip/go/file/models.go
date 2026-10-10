package billing

type Invoice struct {
	Number string
	Lines  []Line
	State  State
}

type Line struct {
	Amount int64
}

type State string

const (
	StateOpen State = "Open"
	StatePaid State = "Paid"
)
