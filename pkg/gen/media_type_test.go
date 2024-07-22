package gen_test

import (
	"bytes"
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/unstoppablemango/tdl/pkg/gen"
	"github.com/unstoppablemango/tdl/pkg/runner"
	"github.com/unstoppablemango/tdl/pkg/uml"
	"google.golang.org/protobuf/proto"
)

var _ = Describe("MediaType", func() {
	DescribeTable("FromMediaType",
		Entry("protobuf", "application/protobuf"),
		Entry("yaml", "application/yaml"),
		Entry("json", "application/json"),
		func(mediaType string) {
			spec := &uml.Spec{Name: "Test"}
			media, err := uml.Marshal(mediaType, spec)
			Expect(err).NotTo(HaveOccurred())

			echo := runner.NewEcho()
			generator := gen.FromMediaType(echo.Gen, mediaType)

			buf := &bytes.Buffer{}
			generator(context.Background(), bytes.NewReader(media), buf)

			actual := &uml.Spec{}
			err = proto.Unmarshal(buf.Bytes(), actual)
			Expect(err).NotTo(HaveOccurred())

			// TODO: Better assertion
			Expect(actual.Name).To(Equal(spec.Name))
		},
	)
})
