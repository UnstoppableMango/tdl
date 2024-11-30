package conform

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	tdl "github.com/unstoppablemango/tdl/pkg"
	"github.com/unstoppablemango/tdl/pkg/testing/e2e"
)

func ItShouldPass(generator tdl.Generator, test *e2e.Test, assertions ...e2e.Assertion) {
	if len(assertions) == 0 {
		panic("test contained no assertions")
	}

	It(fmt.Sprintf("should pass: %s", test.Name), func(ctx context.Context) {
		By("executing the generator")
		output, err := generator.Execute(ctx, test.Spec)

		ExpectWithOffset(1, err).NotTo(HaveOccurred())
		By("performing the given assertions")
		for _, assert := range assertions {
			assert(test, output)
		}
	})
}

func DescribeGenerator(suite e2e.Suite, generator tdl.Generator) {
	for test, assertions := range suite.Tests() {
		ItShouldPass(generator, test, assertions...)
	}
}
