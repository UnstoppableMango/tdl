package matcher

import (
	"errors"
	"fmt"
	"io/fs"
	"reflect"
	"slices"

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

type containFile struct {
	path string
}

// Match implements types.GomegaMatcher.
func (c *containFile) Match(actual interface{}) (success bool, err error) {
	fs, ok := actual.(afero.Fs)
	if !ok {
		return false, fmt.Errorf("expected an [afero.Fs] got %s", reflect.TypeOf(actual))
	}

	_, err = fs.Open(c.path)
	return err == nil, nil
}

// FailureMessage implements types.GomegaMatcher.
func (c *containFile) FailureMessage(actual interface{}) (message string) {
	return fmt.Sprintf("expected file to exist at %s", c.path)
}

// NegatedFailureMessage implements types.GomegaMatcher.
func (c *containFile) NegatedFailureMessage(actual interface{}) (message string) {
	return fmt.Sprintf("expected %s not to exist", c.path)
}

func ContainFile(path string) types.GomegaMatcher {
	return &containFile{path}
}

type beEquivalentToFs struct {
	expected afero.Fs
}

// Match implements types.GomegaMatcher.
func (e *beEquivalentToFs) Match(actual interface{}) (success bool, err error) {
	fsys, ok := actual.(afero.Fs)
	if !ok {
		return false, fmt.Errorf("exected an [afero.Fs] but got %s", reflect.TypeOf(actual))
	}

	failures := []error{}
	err = afero.Walk(e.expected, "",
		func(path string, info fs.FileInfo, err error) error {
			if err != nil {
				return err
			}

			a, err := fsys.Stat(path)
			if err != nil {
				failures = append(failures, err)
				return nil
			}
			if a.IsDir() {
				return nil
			}

			// TODO: How helpful is this really
			if a != info {
				failures = append(failures,
					fmt.Errorf("expected %#v to match %#v", a.Name(), info.Name()),
				)
				return nil
			}

			expectedBytes, err := afero.ReadFile(e.expected, path)
			if err != nil {
				return err
			}

			actualBytes, err := afero.ReadFile(fsys, path)
			if err != nil {
				failures = append(failures,
					fmt.Errorf("expected file at %s to be readable: %w", path, err),
				)
				return nil
			}

			if !slices.Equal(expectedBytes, actualBytes) {
				failures = append(failures,
					fmt.Errorf("expected file at %s to match content:\n\texpected: %s\n\tactual: %s",
						path, string(expectedBytes), string(actualBytes),
					),
				)
				return nil
			}

			return nil
		},
	)
	if err != nil {
		return false, fmt.Errorf("walking expected filesystem: %w", err)
	}

	return len(failures) == 0, errors.Join(failures...)
}

// FailureMessage implements types.GomegaMatcher.
func (e *beEquivalentToFs) FailureMessage(actual interface{}) (message string) {
	return fmt.Sprintf(
		"expected fs %s to match fs %s",
		actual.(afero.Fs).Name(),
		e.expected.Name(),
	)
}

// NegatedFailureMessage implements types.GomegaMatcher.
func (e *beEquivalentToFs) NegatedFailureMessage(actual interface{}) (message string) {
	return fmt.Sprintf(
		"expected fs %s not to match fs %s",
		actual.(afero.Fs).Name(),
		e.expected.Name(),
	)
}

func BeEquivalentToFs(fs afero.Fs) types.GomegaMatcher {
	return &beEquivalentToFs{fs}
}
