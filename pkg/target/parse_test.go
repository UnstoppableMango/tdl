package target_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/unstoppablemango/tdl/pkg/target"
)

var _ = Describe("Parse", func() {
	DescribeTable("TypeScript",
		Entry(nil, "ts"),
		Entry(nil, "TS"),
		Entry(nil, "typescript"),
		Entry(nil, "tyPEsCrIpt"),
		func(input string) {
			result, err := target.Parse(input)

			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(BeAssignableToTypeOf(&target.TypeScript{}))
		},
	)

	DescribeTable("Unsupported",
		Entry(nil, "dfkjsalkdfjksdl"),
		func(input string) {
			_, err := target.Parse(input)

			Expect(err).To(MatchError(target.ErrUnsupported))
		},
	)
})