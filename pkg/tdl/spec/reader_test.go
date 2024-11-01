package spec_test

import (
	"io"
	"testing/quick"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"google.golang.org/protobuf/proto"

	"github.com/unstoppablemango/tdl/pkg/tdl"
	"github.com/unstoppablemango/tdl/pkg/tdl/spec"
	"github.com/unstoppablemango/tdl/pkg/testing"
	tdlv1alpha1 "github.com/unstoppablemango/tdl/pkg/unmango/dev/tdl/v1alpha1"
)

var _ = Describe("Reader", func() {
	It("should create a new reader", func() {
		r := spec.NewReader(&tdlv1alpha1.Spec{})

		Expect(r).NotTo(BeNil())
	})

	It("should return n == 0 when len(p) == 0", func() {
		r := spec.NewReader(&tdlv1alpha1.Spec{})

		n, err := r.Read([]byte{})

		Expect(err).NotTo(HaveOccurred())
		Expect(n).To(BeZero())
	})

	It("should read protobuf by default", func() {
		fn := func(name, displayName string) bool {
			s := &tdlv1alpha1.Spec{Name: name, DisplayName: displayName}
			r := spec.NewReader(s)

			data, err := io.ReadAll(r)
			Expect(err).NotTo(HaveOccurred())
			a, err := spec.FromProto(data)
			Expect(err).NotTo(HaveOccurred())

			return proto.Equal(a, s)
		}

		Expect(quick.Check(fn, nil)).To(Succeed())
	})

	DescribeTable("MediaType",
		testing.MediaTypeEntries(),
		func(media tdl.MediaType) {
			fn := func(name, displayName string) bool {
				s := &tdlv1alpha1.Spec{Name: name, DisplayName: displayName}
				r := spec.NewReader(s, spec.WithMediaType(media))

				data, err := io.ReadAll(r)
				Expect(err).NotTo(HaveOccurred())
				a, err := spec.FromMediaType(media, data)
				Expect(err).NotTo(HaveOccurred())

				return proto.Equal(a, s)
			}

			Expect(quick.Check(fn, nil)).To(Succeed())
		},
	)
})
