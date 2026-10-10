package shop

import (
	"errors"
	"fmt"
	"regexp"
	"unicode/utf8"
)

// An address to write to.
type Email string

var patternEmail_0 = regexp.MustCompile(`^[^@]+@[^@]+$`)

// Validate reports every constraint Email breaks, joined, or nil.
func (e Email) Validate() error {
	return errors.Join(e.validate("Email", nil)...)
}

func (e Email) validate(path string, errs []error) []error {
	if !patternEmail_0.MatchString(string(e)) {
		errs = append(errs, fmt.Errorf("%s: matches(/^[^@]+@[^@]+$/): no match", path))
	}
	if count := utf8.RuneCountInString(string(e)); count < 3 || count > 254 {
		errs = append(errs, fmt.Errorf("%s: length(3..254): got %d", path, count))
	}
	return errs
}
