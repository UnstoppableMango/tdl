package spec_test

import (
	"io"
	"testing/quick"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"google.golang.org/protobuf/proto"

	"github.com/unstoppablemango/tdl/pkg/tdl/spec"
	tdlv1alpha1 "github.com/unstoppablemango/tdl/pkg/unmango/dev/tdl/v1alpha1"
)

var _ = Describe("Reader", func() {
	It("should create a new reader", func() {
		r := spec.NewReader(&tdlv1alpha1.Spec{})

		Expect(r).NotTo(BeNil())
	})

	It("should read protobuf by default", Pending, func() {
		fn := func(name string) bool {
			s := &tdlv1alpha1.Spec{Name: name}
			r := spec.NewReader(s)

			data, err := io.ReadAll(r)
			Expect(err).NotTo(HaveOccurred())
			a, err := spec.FromProto(data)
			Expect(err).NotTo(HaveOccurred())

			return proto.Equal(a, s)
		}

		Expect(quick.Check(fn, nil)).To(Succeed())
	})
})
