package mediatype_test

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/unstoppablemango/tdl/pkg/mediatype"
	"github.com/unstoppablemango/tdl/pkg/tdl"
)

var _ = Describe("Mediatype", func() {
	DescribeTable("Parse",
		Entry(fmt.Sprintf("should parse %s", mediatype.ApplicationGoogleProtobuf), "application/vnd.google.protobuf"),
		Entry(fmt.Sprintf("should parse %s", mediatype.ApplicationJson), "application/json"),
		Entry(fmt.Sprintf("should parse %s", mediatype.ApplicationProtobuf), "application/protobuf"),
		Entry(fmt.Sprintf("should parse %s", mediatype.ApplicationXProtobuf), "application/x-protobuf"),
		Entry(fmt.Sprintf("should parse %s", mediatype.ApplicationXYaml), "application/x-yaml"),
		Entry(fmt.Sprintf("should parse %s", mediatype.ApplicationYaml), "application/yaml"),
		Entry(fmt.Sprintf("should parse %s", mediatype.TextJson), "text/json"),
		Entry(fmt.Sprintf("should parse %s", mediatype.TextYaml), "text/yaml"),
		func(value string) {
			Expect(mediatype.Parse(value)).NotTo(BeEmpty())
		},
	)

	DescribeTable("Parse unsupported",
		Entry(fmt.Sprintf("should not parse %s", "vnd.google.protobuf"), "vnd.google.protobuf"),
		Entry(fmt.Sprintf("should not parse %s", "application/jsno"), "application/jsno"),
		Entry(fmt.Sprintf("should not parse %s", "aplication/protobuf"), "aplication/protobuf"),
		Entry(fmt.Sprintf("should not parse %s", "application/xprotobuf"), "application/xprotobuf"),
		Entry(fmt.Sprintf("should not parse %s", "application/x-yml"), "application/x-yml"),
		Entry(fmt.Sprintf("should not parse %s", "application/yml"), "application/yml"),
		Entry(fmt.Sprintf("should not parse %s", "txt/json"), "txt/json"),
		Entry(fmt.Sprintf("should not parse %s", "textyaml"), "textyaml"),
		func(value string) {
			_, err := mediatype.Parse(value)

			Expect(err).To(HaveOccurred())
		},
	)

	DescribeTable("Supported",
		Entry(fmt.Sprintf("should support %s", mediatype.ApplicationGoogleProtobuf), mediatype.ApplicationGoogleProtobuf),
		Entry(fmt.Sprintf("should support %s", mediatype.ApplicationJson), mediatype.ApplicationJson),
		Entry(fmt.Sprintf("should support %s", mediatype.ApplicationProtobuf), mediatype.ApplicationProtobuf),
		Entry(fmt.Sprintf("should support %s", mediatype.ApplicationXProtobuf), mediatype.ApplicationXProtobuf),
		Entry(fmt.Sprintf("should support %s", mediatype.ApplicationXYaml), mediatype.ApplicationXYaml),
		Entry(fmt.Sprintf("should support %s", mediatype.ApplicationYaml), mediatype.ApplicationYaml),
		Entry(fmt.Sprintf("should support %s", mediatype.TextJson), mediatype.TextJson),
		Entry(fmt.Sprintf("should support %s", mediatype.TextYaml), mediatype.TextYaml),
		func(value tdl.MediaType) {
			Expect(mediatype.Supported(value)).To(BeTrue())
		},
	)
})
