package matcher

import (
	"fmt"

	"github.com/onsi/gomega/types"
)

type lenGreaterThanZero struct{}

// Match implements types.GomegaMatcher.
func (l lenGreaterThanZero) Match(actual interface{}) (success bool, err error) {
	if s, ok := actual.([]byte); ok {
		return len(s) > 0, nil
	}
	if s, ok := actual.([]int); ok {
		return len(s) > 0, nil
	}
	if s, ok := actual.([]error); ok {
		return len(s) > 0, nil
	}

	return false, fmt.Errorf("not a slice: %#v", actual)
}

// FailureMessage implements types.GomegaMatcher.
func (l lenGreaterThanZero) FailureMessage(actual interface{}) (message string) {
	return fmt.Sprintf("Expected\n\t%#v\n to have\n\tlen > 0", actual)
}

// NegatedFailureMessage implements types.GomegaMatcher.
func (l lenGreaterThanZero) NegatedFailureMessage(actual interface{}) (message string) {
	return fmt.Sprintf("Expected\n\t%#v\n to have\n\tlen <= 0", actual)
}

func HaveLenGreaterThanZero() types.GomegaMatcher {
	return lenGreaterThanZero{}
}
