package conform

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/unmango/go/iter"
	"github.com/unmango/go/slices"
	tdl "github.com/unstoppablemango/tdl/pkg"
	"github.com/unstoppablemango/tdl/pkg/testing/e2e"
	. "github.com/unstoppablemango/tdl/pkg/testing/matcher"
)

type Suite interface {
	ConstructTestsFor(tdl.Generator)
}

type suite struct {
	tests iter.Seq[*e2e.Test]
}

// ConstructTestsFor implements Suite.
func (s *suite) ConstructTestsFor(generator tdl.Generator) {
	for test := range s.tests {
		ItShouldPass(generator, test)
	}
}

func NewSuite(name string, tests ...*e2e.Test) Suite {
	if len(tests) == 0 {
		panic("no tests defined")
	}

	return &suite{slices.Values(tests)}
}

func IncludeTests(s e2e.Suite) Suite {
	return &suite{s.Tests()}
}

func ItShouldPass(generator tdl.Generator, test *e2e.Test) {
	It(fmt.Sprintf("should pass: %s", test.Name), func(ctx context.Context) {
		output, err := generator.Execute(ctx, test.Spec)

		Expect(err).NotTo(HaveOccurred())
		Expect(output).To(BeEquivalentToFs(test.Expected))
	})
}
