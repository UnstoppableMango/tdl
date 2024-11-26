package input_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/unstoppablemango/tdl/pkg/input"
)

var _ = Describe("Parse", func() {
	It("should work", func() {
		res, err := input.ParseArgs(nil, []string{})

		Expect(err).NotTo(HaveOccurred())
		Expect(res.Inputs).To(HaveLen(1))
	})
})
