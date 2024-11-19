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
	Execute(tdl.Generator)
}

type suite struct {
	tests []*testing.Test
}

// Execute implements Suite.
func (s *suite) Execute(generator tdl.Generator) {
	for _, test := range s.tests {
		It(fmt.Sprintf("should pass: %s", test.Name), func(ctx context.Context) {
			output, err := generator.Execute(ctx, test.Spec)

			Expect(err).NotTo(HaveOccurred())
			Expect(output).To(BeEquivalentToFs(test.Expected))
		})
	}
}

func NewSuite(tests ...*testing.Test) Suite {
	if len(tests) == 0 {
		panic("no tests defined")
	}

	return &suite{tests}
}
