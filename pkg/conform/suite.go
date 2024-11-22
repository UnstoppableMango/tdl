package conform

import (
	"context"
	"fmt"

	"github.com/charmbracelet/log"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/unmango/go/iter"
	"github.com/unmango/go/slices"
	tdl "github.com/unstoppablemango/tdl/pkg"
	"github.com/unstoppablemango/tdl/pkg/testing/e2e"
)

type Suite interface {
	ConstructTests(tdl.Generator)
	Expect(string, ...e2e.Assertion)
}

type suite struct {
	tests      iter.Seq[*e2e.Test]
	assertions map[string][]e2e.Assertion
}

// ConstructTests implements Suite.
func (s *suite) ConstructTests(generator tdl.Generator) {
	for test := range s.tests {
		assertions := s.assertions[test.Name]
		ItShouldPass(generator, test, assertions...)
	}
}

func (s suite) Expect(name string, assertions ...e2e.Assertion) {
	s.assertions[name] = append(s.assertions[name], assertions...)
}

func NewSuite(name string, tests ...*e2e.Test) Suite {
	if len(tests) == 0 {
		panic("no tests defined")
	}

	return &suite{tests: slices.Values(tests)}
}

func IncludeTests(s e2e.Suite) Suite {
	return &suite{tests: s.Tests()}
}

func ItShouldPass(generator tdl.Generator, test *e2e.Test, assertions ...e2e.Assertion) {
	It(fmt.Sprintf("should pass: %s", test.Name), func(ctx context.Context) {
		log.SetLevel(log.DebugLevel)
		output, err := generator.Execute(ctx, test.Spec)

		Expect(err).NotTo(HaveOccurred())
		for _, assert := range assertions {
			assert(test, output)
		}
		if len(assertions) == 0 {
			log.New(GinkgoWriter).Warnf("no assertions for: %s", test.Name)
		}
	})
}
