package conform

import (
	"bytes"
	"fmt"

	"github.com/onsi/ginkgo/v2"
	g "github.com/onsi/gomega"
	"github.com/unstoppablemango/tdl/pkg/pipe"
	"github.com/unstoppablemango/tdl/pkg/testing"
)

// IOSuite is a helper function for defining IOTests.
// It calls IOTest for each testing.Test in tests
//
// IOSuite MUST be called during the Ginkgo test construction phase
func IOSuite(tests []*testing.Test, pipeline pipe.IO) {
	for _, test := range tests {
		_ = IOTest(test, pipeline)
	}
}

// IOTest asserts that given [test.Input] [generator] produces [test.Output]
func IOTest(test *testing.Test, pipeline pipe.IO) bool {
	return ginkgo.It(fmt.Sprintf("should pass: %s", test.Name), func() {
		expected := string(test.Output)
		input := bytes.NewReader(test.Input)
		output := &bytes.Buffer{}

		err := pipeline(input, output)
		actual := output.String()

		g.Expect(err).NotTo(g.HaveOccurred(), actual)
		g.Expect(actual).To(g.Equal(expected))
	})
}