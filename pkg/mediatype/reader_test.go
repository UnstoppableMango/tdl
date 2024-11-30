package mediatype_test

import (
	"encoding/json"
	"io"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"google.golang.org/protobuf/proto"
	"gopkg.in/yaml.v3"

	tdl "github.com/unstoppablemango/tdl/pkg"
	"github.com/unstoppablemango/tdl/pkg/mediatype"
	. "github.com/unstoppablemango/tdl/pkg/testing/matcher"
	tdlv1alpha1 "github.com/unstoppablemango/tdl/pkg/unmango/dev/tdl/v1alpha1"
)

var _ = Describe("Reader", func() {
	DescribeTable("JSON",
		Entry(nil, mediatype.ApplicationJson),
		Entry(nil, mediatype.TextJson),
		func(media tdl.MediaType) {
			s := &tdlv1alpha1.Spec{Name: "testing"}

			r := mediatype.ProtoReader(s, media)

			data, err := io.ReadAll(r)
			Expect(err).NotTo(HaveOccurred())
			Expect(data).To(HaveLenGreaterThanZero())
			expected, err := json.Marshal(s)
			Expect(err).NotTo(HaveOccurred())
			Expect(data).To(Equal(expected))
		},
	)

	DescribeTable("Protobuf",
		Entry(nil, mediatype.ApplicationGoogleProtobuf),
		Entry(nil, mediatype.ApplicationProtobuf),
		Entry(nil, mediatype.ApplicationXProtobuf),
		func(media tdl.MediaType) {
			s := &tdlv1alpha1.Spec{Name: "testing"}

			r := mediatype.ProtoReader(s, media)

			data, err := io.ReadAll(r)
			Expect(err).NotTo(HaveOccurred())
			Expect(data).To(HaveLenGreaterThanZero())
			expected, err := proto.Marshal(s)
			Expect(err).NotTo(HaveOccurred())
			Expect(data).To(Equal(expected))
		},
	)

	DescribeTable("Yaml",
		Entry(nil, mediatype.ApplicationXYaml),
		Entry(nil, mediatype.ApplicationYaml),
		Entry(nil, mediatype.TextYaml),
		func(media tdl.MediaType) {
			s := &tdlv1alpha1.Spec{Name: "testing"}

			r := mediatype.ProtoReader(s, media)

			data, err := io.ReadAll(r)
			Expect(err).NotTo(HaveOccurred())
			Expect(data).To(HaveLenGreaterThanZero())
			expected, err := yaml.Marshal(s)
			Expect(err).NotTo(HaveOccurred())
			Expect(data).To(Equal(expected))
		},
	)
})
