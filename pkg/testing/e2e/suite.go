package e2e

import (
	"fmt"
	"path/filepath"

	"github.com/charmbracelet/log"
	"github.com/spf13/afero"
	"github.com/unmango/go/iter"
	"github.com/unmango/go/option"
)

type Options struct {
	assertions map[string][]Assertion
}

type (
	Option    func(*Options)
	Assertion func(*Test, afero.Fs)
)

type Suite interface {
	Name() string
	Tests() iter.Seq2[*Test, []Assertion]
}

type suite struct {
	Options
	name  string
	tests iter.Seq[*Test]
}

// Name implements Suite.
func (s suite) Name() string {
	return s.name
}

// Tests implements Suite.
func (s suite) Tests() iter.Seq2[*Test, []Assertion] {
	return func(yield func(*Test, []Assertion) bool) {
		for t := range s.tests {
			assertions, ok := s.assertions[t.Name]
			if !ok || len(assertions) == 0 {
				log.Warnf("no assertions for test: %s", t.Name)
			}

			if !yield(t, assertions) {
				break
			}
		}
	}
}

func ReadSuite(fs afero.Fs, path string, options ...Option) (Suite, error) {
	tests, err := ListTests(fs, path)
	if err != nil {
		return nil, fmt.Errorf("reading suite: %w", err)
	}

	suite := suite{
		name:  filepath.Base(path),
		tests: tests,
	}
	option.ApplyAll(&suite.Options, options)

	return suite, nil
}

func ReadTests(fs afero.Fs, path string, assertions map[string][]Assertion) (Suite, error) {
	tests := iter.Empty[*Test]()
	for name := range assertions {
		test, err := ReadTest(fs, filepath.Join(path, name))
		if err != nil {
			return nil, fmt.Errorf("reading test %s: %w", name, err)
		}

		tests = iter.Append(tests, test)
	}

	return suite{
		name:  filepath.Base(path),
		tests: tests,
		Options: Options{
			assertions: assertions,
		},
	}, nil
}

func Expect(name string, assertions ...Assertion) Option {
	return func(o *Options) {
		existing, ok := o.assertions[name]
		if !ok {
			existing = []Assertion{}
		}

		o.assertions[name] = append(existing, assertions...)
	}
}
