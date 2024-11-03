package mediatype_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/unstoppablemango/tdl/pkg/mediatype"
	"github.com/unstoppablemango/tdl/pkg/tdl"
)

var _ = Describe("Match", func() {
	DescribeTable("Match protobuf",
		Entry(nil, mediatype.ApplicationGoogleProtobuf),
		Entry(nil, mediatype.ApplicationProtobuf),
		Entry(nil, mediatype.ApplicationXProtobuf),
		func(media tdl.MediaType) {
			sentinel := false
			matcher := mediatype.Matcher[any]{
				Protobuf: func() (any, error) {
					sentinel = true
					return nil, nil
				},
			}

			_, err := mediatype.Match(media, matcher)

			Expect(err).NotTo(HaveOccurred())
			Expect(sentinel).To(BeTrueBecause("protobuf was matched"))
		},
	)

	DescribeTable("Match JSON",
		Entry(nil, mediatype.ApplicationJson),
		Entry(nil, mediatype.TextJson),
		func(media tdl.MediaType) {
			sentinel := false
			matcher := mediatype.Matcher[any]{
				Json: func() (any, error) {
					sentinel = true
					return nil, nil
				},
			}

			_, err := mediatype.Match(media, matcher)

			Expect(err).NotTo(HaveOccurred())
			Expect(sentinel).To(BeTrueBecause("JSON was matched"))
		},
	)

	DescribeTable("Match yaml",
		Entry(nil, mediatype.ApplicationXYaml),
		Entry(nil, mediatype.ApplicationYaml),
		Entry(nil, mediatype.TextYaml),
		func(media tdl.MediaType) {
			sentinel := false
			matcher := mediatype.Matcher[any]{
				Yaml: func() (any, error) {
					sentinel = true
					return nil, nil
				},
			}

			_, err := mediatype.Match(media, matcher)

			Expect(err).NotTo(HaveOccurred())
			Expect(sentinel).To(BeTrueBecause("yaml was matched"))
		},
	)

	DescribeTable("Match other",
		Entry(nil, tdl.MediaType("other")),
		Entry(nil, tdl.MediaType("text")),
		Entry(nil, tdl.MediaType("yml")),
		func(media tdl.MediaType) {
			sentinel := false
			matcher := mediatype.Matcher[any]{
				Other: func() (any, error) {
					sentinel = true
					return nil, nil
				},
			}

			_, err := mediatype.Match(media, matcher)

			Expect(err).NotTo(HaveOccurred())
			Expect(sentinel).To(BeTrueBecause("other was matched"))
		},
	)
})
