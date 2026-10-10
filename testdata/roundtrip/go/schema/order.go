// Hand-maintained, in the shape tdl writes.

package shop

import (
	"errors"
	"fmt"
	"time"
)

// An order, identified by its id.
type Order struct {
	Id string `json:"id"`
	// When it was placed.
	PlacedAt time.Time
	Items    []LineItem
	Tags     map[string]struct{}
	Note     *string
	Contact  Email
	Status   Status
	Payment  *Payment
	// Deprecated: nobody has one
	Fax     *string
	Attrs   map[string]int64
	Took    time.Duration
	Receipt []byte
}

// Key returns what identifies this Order.
func (o Order) Key() string {
	return o.Id
}

// Validate reports every constraint Order breaks, joined, or nil.
func (o Order) Validate() error {
	return errors.Join(o.validate("Order", nil)...)
}

func (o Order) validate(path string, errs []error) []error {
	if count := len(o.Items); count < 1 {
		errs = append(errs, fmt.Errorf("%s.items: length(1..): got %d", path, count))
	}
	for idx, elem := range o.Items {
		errs = elem.validate(fmt.Sprintf("%s.items[%d]", path, idx), errs)
	}
	errs = o.Contact.validate(path+".contact", errs)
	if o.Payment != nil {
		if inner, ok := (*o.Payment).(interface{ validate(string, []error) []error }); ok {
			errs = inner.validate(path+".payment", errs)
		}
	}
	return errs
}
