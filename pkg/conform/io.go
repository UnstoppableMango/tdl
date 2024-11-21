package conform

import (
	"bytes"
	"context"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/unstoppablemango/tdl/pkg/pipe"
	"github.com/unstoppablemango/tdl/pkg/testing"
)

// IOSuite is a helper function for defining IOTests.
// It calls [IOTest] for each testing.Test in tests
//
// [IOSuite] MUST be called during the Ginkgo test construction phase
func IOSuite(tests []*testing.RawTest, pipeline pipe.IO) {
	for _, test := range tests {
		_ = IOTest(test, pipeline)
	}
}

// IOTest asserts that given [test.Input] [generator] produces [test.Output]
func IOTest(test *testing.RawTest, pipeline pipe.IO) bool {
	return It(fmt.Sprintf("should pass: %s", test.Name), func(ctx context.Context) {
		expected := string(test.Output)
		input := bytes.NewReader(test.Input)
		output := &bytes.Buffer{}

		err := pipeline(ctx, input, output)
		actual := output.String()

		Expect(err).NotTo(HaveOccurred(), actual)
		Expect(actual).To(Equal(expected))
	})
}
