package conform

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	tdl "github.com/unstoppablemango/tdl/pkg"
	"github.com/unstoppablemango/tdl/pkg/testing"
	. "github.com/unstoppablemango/tdl/pkg/testing/matcher"
)

type Suite interface {
	Describe(tdl.Generator)
}

type suite struct {
	tests []*testing.Test
}

// Describe implements Suite.
func (s *suite) Describe(generator tdl.Generator) {
	Describe(fmt.Sprintf("%s Suite", generator), func() {
		for _, test := range s.tests {
			DescribeTest(test, generator)
		}
	})
}

func NewSuite(tests ...*testing.Test) Suite {
	if len(tests) == 0 {
		panic("no tests defined")
	}

	return &suite{tests}
}

func DescribeTest(test *testing.Test, generator tdl.Generator) {
	It(fmt.Sprintf("should pass: %s", test.Name), func(ctx context.Context) {
		output, err := generator.Execute(ctx, test.Spec)

		Expect(err).NotTo(HaveOccurred())
		Expect(output).To(BeEquivalentToFs(test.Expected))
	})
}
