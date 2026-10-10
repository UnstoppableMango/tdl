package shop

import (
	"errors"
	"fmt"
)

type LineItem struct {
	Order    string
	Sku      string // what was bought
	Quantity int64
}

// LineItemKey is what identifies a LineItem.
type LineItemKey struct {
	Order string
	Sku   string
}

// Key returns what identifies this LineItem.
func (l LineItem) Key() LineItemKey {
	return LineItemKey{Order: l.Order, Sku: l.Sku}
}

// Validate reports every constraint LineItem breaks, joined, or nil.
func (l LineItem) Validate() error {
	return errors.Join(l.validate("LineItem", nil)...)
}

func (l LineItem) validate(path string, errs []error) []error {
	if l.Quantity < 1 {
		errs = append(errs, fmt.Errorf("%s.quantity: min(1): got %d", path, l.Quantity))
	}
	return errs
}
