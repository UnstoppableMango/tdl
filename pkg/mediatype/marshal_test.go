package mediatype_test

import (
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"google.golang.org/protobuf/proto"
	"gopkg.in/yaml.v3"

	tdl "github.com/unstoppablemango/tdl/pkg"
	"github.com/unstoppablemango/tdl/pkg/mediatype"
	. "github.com/unstoppablemango/tdl/pkg/testing/matcher"
	tdlv1alpha1 "github.com/unstoppablemango/tdl/pkg/unmango/dev/tdl/v1alpha1"
)

var _ = Describe("Marshal", func() {
	DescribeTable("JSON",
		Entry(nil, mediatype.ApplicationJson),
		Entry(nil, mediatype.TextJson),
		func(media tdl.MediaType) {
			s := &tdlv1alpha1.Spec{Name: "testing"}

			data, err := mediatype.Marshal(s, media)

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

			data, err := mediatype.Marshal(s, media)

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

			data, err := mediatype.Marshal(s, media)

			Expect(err).NotTo(HaveOccurred())
			Expect(data).To(HaveLenGreaterThanZero())
			expected, err := yaml.Marshal(s)
			Expect(err).NotTo(HaveOccurred())
			Expect(data).To(Equal(expected))
		},
	)

	Describe("Unmarhsal", func() {
		DescribeTable("JSON",
			Entry(nil, mediatype.ApplicationJson),
			Entry(nil, mediatype.TextJson),
			func(media tdl.MediaType) {
				expected := &tdlv1alpha1.Spec{Name: "testing"}
				data, err := json.Marshal(expected)
				Expect(err).NotTo(HaveOccurred())
				result := &tdlv1alpha1.Spec{}

				err = mediatype.Unmarshal(data, result, media)

				Expect(err).NotTo(HaveOccurred())
				Expect(result).To(EqualProto(expected))
			},
		)

		DescribeTable("Protobuf",
			Entry(nil, mediatype.ApplicationGoogleProtobuf),
			Entry(nil, mediatype.ApplicationProtobuf),
			Entry(nil, mediatype.ApplicationXProtobuf),
			func(media tdl.MediaType) {
				expected := &tdlv1alpha1.Spec{Name: "testing"}
				data, err := proto.Marshal(expected)
				Expect(err).NotTo(HaveOccurred())
				result := &tdlv1alpha1.Spec{}

				err = mediatype.Unmarshal(data, result, media)

				Expect(err).NotTo(HaveOccurred())
				Expect(result).To(EqualProto(expected))
			},
		)

		DescribeTable("Yaml",
			Entry(nil, mediatype.ApplicationXYaml),
			Entry(nil, mediatype.ApplicationYaml),
			Entry(nil, mediatype.TextYaml),
			func(media tdl.MediaType) {
				expected := &tdlv1alpha1.Spec{Name: "testing"}
				data, err := yaml.Marshal(expected)
				Expect(err).NotTo(HaveOccurred())
				result := &tdlv1alpha1.Spec{}

				err = mediatype.Unmarshal(data, result, media)

				Expect(err).NotTo(HaveOccurred())
				Expect(result).To(EqualProto(expected))
			},
		)
	})
})
