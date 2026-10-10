package shop

// Where an order is.
type Status string

const (
	// Not yet sent.
	StatusDraft  Status = "Draft"
	StatusPlaced Status = "Placed"
	StatusLow    Status = "Lowest"
)
