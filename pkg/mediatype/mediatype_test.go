package mediatype_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/unstoppablemango/tdl/pkg/mediatype"
	"github.com/unstoppablemango/tdl/pkg/tdl"
)

var _ = Describe("Mediatype", func() {
	DescribeTable("Parse",
		Entry(nil, "application/vnd.google.protobuf"),
		Entry(nil, "application/json"),
		Entry(nil, "application/protobuf"),
		Entry(nil, "application/x-protobuf"),
		Entry(nil, "application/x-yaml"),
		Entry(nil, "application/yaml"),
		Entry(nil, "text/json"),
		Entry(nil, "text/yaml"),
		func(value string) {
			Expect(mediatype.Parse(value)).NotTo(BeEmpty())
		},
	)

	DescribeTable("Parse unsupported",
		Entry(nil, "vnd.google.protobuf"),
		Entry(nil, "application/jsno"),
		Entry(nil, "aplication/protobuf"),
		Entry(nil, "application/xprotobuf"),
		Entry(nil, "application/x-yml"),
		Entry(nil, "application/yml"),
		Entry(nil, "txt/json"),
		Entry(nil, "textyaml"),
		func(value string) {
			_, err := mediatype.Parse(value)

			Expect(err).To(HaveOccurred())
		},
	)

	DescribeTable("Supported",
		Entry(nil, mediatype.ApplicationGoogleProtobuf),
		Entry(nil, mediatype.ApplicationJson),
		Entry(nil, mediatype.ApplicationProtobuf),
		Entry(nil, mediatype.ApplicationXProtobuf),
		Entry(nil, mediatype.ApplicationXYaml),
		Entry(nil, mediatype.ApplicationYaml),
		Entry(nil, mediatype.TextJson),
		Entry(nil, mediatype.TextYaml),
		func(value tdl.MediaType) {
			Expect(mediatype.Supported(value)).To(BeTrue())
		},
	)
})
