package mediatype_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	tdl "github.com/unstoppablemango/tdl/pkg"
	"github.com/unstoppablemango/tdl/pkg/mediatype"
)

var _ = Describe("Guess", func() {
	DescribeTable("known extensions",
		Entry(nil, "thing.yaml", mediatype.ApplicationYaml),
		Entry(nil, "thing.yml", mediatype.ApplicationYaml),
		Entry(nil, "some/path/thing.yml", mediatype.ApplicationYaml),
		Entry(nil, "thing.json", mediatype.ApplicationJson),
		Entry(nil, "thing.pb", mediatype.ApplicationProtobuf),
		func(token string, expected tdl.MediaType) {
			result, err := mediatype.Guess(token)

			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(expected))
		},
	)

	DescribeTable("unknown extensions",
		Entry(nil, "thing.yl"),
		Entry(nil, "thing.proto"),
		func(token string) {
			_, err := mediatype.Guess(token)

			Expect(err).To(HaveOccurred())
		},
	)
})
