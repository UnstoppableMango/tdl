package matcher

import (
	"fmt"
	"reflect"

	"github.com/onsi/gomega/types"
	"github.com/spf13/afero"
)

type containFileWithBytes struct {
	path  string
	bytes []byte
}

// Match implements types.GomegaMatcher.
func (c *containFileWithBytes) Match(actual interface{}) (success bool, err error) {
	fs, ok := actual.(afero.Fs)
	if !ok {
		return false, fmt.Errorf("expected an [afero.Fs] got %s", reflect.TypeOf(actual))
	}

	return afero.FileContainsBytes(fs, c.path, c.bytes)
}

// FailureMessage implements types.GomegaMatcher.
func (c *containFileWithBytes) FailureMessage(actual interface{}) (message string) {
	return fmt.Sprintf("expected file at %s to contain bytes %#v", c.path, c.bytes)
}

// NegatedFailureMessage implements types.GomegaMatcher.
func (c *containFileWithBytes) NegatedFailureMessage(actual interface{}) (message string) {
	return fmt.Sprintf("expected file at %s not to contain bytes %#v", c.path, c.bytes)
}

func ContainFileWithBytes(path string, bytes []byte) types.GomegaMatcher {
	return &containFileWithBytes{path, bytes}
}
