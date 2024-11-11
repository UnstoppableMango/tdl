package progress

type Event struct {
	N int
}

type ReportFunc func(int, error)
