package media_test

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/unstoppablemango/tdl/pkg/tdl"
	"github.com/unstoppablemango/tdl/pkg/tdl/media"
)

var _ = Describe("Mediatype", func() {
	DescribeTable("Parse",
		Entry(fmt.Sprintf("should parse %s", media.ApplicationGoogleProtobuf), "application/vnd.google.protobuf"),
		Entry(fmt.Sprintf("should parse %s", media.ApplicationJson), "application/json"),
		Entry(fmt.Sprintf("should parse %s", media.ApplicationProtobuf), "application/protobuf"),
		Entry(fmt.Sprintf("should parse %s", media.ApplicationXProtobuf), "application/x-protobuf"),
		Entry(fmt.Sprintf("should parse %s", media.ApplicationXYaml), "application/x-yaml"),
		Entry(fmt.Sprintf("should parse %s", media.ApplicationYaml), "application/yaml"),
		Entry(fmt.Sprintf("should parse %s", media.TextJson), "text/json"),
		Entry(fmt.Sprintf("should parse %s", media.TextYaml), "text/yaml"),
		func(value string) {
			Expect(media.Parse(value)).NotTo(BeEmpty())
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
			_, err := media.Parse(value)

			Expect(err).To(HaveOccurred())
		},
	)

	DescribeTable("Supported",
		Entry(fmt.Sprintf("should support %s", media.ApplicationGoogleProtobuf), media.ApplicationGoogleProtobuf),
		Entry(fmt.Sprintf("should support %s", media.ApplicationJson), media.ApplicationJson),
		Entry(fmt.Sprintf("should support %s", media.ApplicationProtobuf), media.ApplicationProtobuf),
		Entry(fmt.Sprintf("should support %s", media.ApplicationXProtobuf), media.ApplicationXProtobuf),
		Entry(fmt.Sprintf("should support %s", media.ApplicationXYaml), media.ApplicationXYaml),
		Entry(fmt.Sprintf("should support %s", media.ApplicationYaml), media.ApplicationYaml),
		Entry(fmt.Sprintf("should support %s", media.TextJson), media.TextJson),
		Entry(fmt.Sprintf("should support %s", media.TextYaml), media.TextYaml),
		func(value tdl.MediaType) {
			Expect(media.Supported(value)).To(BeTrue())
		},
	)
})
