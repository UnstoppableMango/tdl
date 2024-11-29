package run_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/unstoppablemango/tdl/pkg/config/run"
)

var _ = Describe("Parse", func() {
	It("should parse stdin", func() {
		config, err := run.ParseArgs([]string{"-"})

		Expect(err).NotTo(HaveOccurred())
		Expect(config.Inputs).To(HaveLen(1))
		input := config.Inputs[0]
		Expect(input.GetStdin()).To(BeTrue())
	})
})
