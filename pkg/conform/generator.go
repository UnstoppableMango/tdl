package conform

import (
	"context"
	"fmt"

	"github.com/charmbracelet/log"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	tdl "github.com/unstoppablemango/tdl/pkg"
	"github.com/unstoppablemango/tdl/pkg/testing/e2e"
)

type GeneratorSuite interface {
	DescribeGenerator(tdl.Generator)
}

type generatorSuite struct{ e2e.Suite }

func (s *generatorSuite) DescribeGenerator(generator tdl.Generator) {
	for test, assertions := range s.Tests() {
		ItShouldPass(generator, test, assertions...)
	}
}

func ItShouldPass(generator tdl.Generator, test *e2e.Test, assertions ...e2e.Assertion) {
	It(fmt.Sprintf("should pass: %s", test.Name), func(ctx context.Context) {
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
