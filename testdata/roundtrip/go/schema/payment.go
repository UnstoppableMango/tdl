package shop

import (
	"errors"
	"fmt"
	"unicode/utf8"
)

// How an order is paid.
type Payment interface {
	Priced
	isPayment()
}

type PaymentCard struct {
	Last4 string
}

func (PaymentCard) isPayment() {}

func (PaymentCard) isPriced() {}

// Validate reports every constraint Payment.Card breaks, joined, or nil.
func (p PaymentCard) Validate() error {
	return errors.Join(p.validate("Payment.Card", nil)...)
}

func (p PaymentCard) validate(path string, errs []error) []error {
	if count := utf8.RuneCountInString(p.Last4); count != 4 {
		errs = append(errs, fmt.Errorf("%s.last4: length(4): got %d", path, count))
	}
	return errs
}

type PaymentCash struct {
}

func (PaymentCash) isPayment() {}

func (PaymentCash) isPriced() {}
