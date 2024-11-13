package progress

type Event struct {
	Percent float64
}

type ReportFunc func(float64, error)
