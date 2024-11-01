package spec_test

import (
	"testing/quick"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"google.golang.org/protobuf/proto"

	"github.com/unstoppablemango/tdl/pkg/tdl"
	"github.com/unstoppablemango/tdl/pkg/tdl/spec"
	"github.com/unstoppablemango/tdl/pkg/testing"
	tdlv1alpha1 "github.com/unstoppablemango/tdl/pkg/unmango/dev/tdl/v1alpha1"
)

var _ = Describe("Parse", func() {
	Describe("JSON", func() {
		It("should round-trip", func() {
			fn := func(name string) bool {
				s := &tdlv1alpha1.Spec{Name: name}

				data, err := spec.ToJson(s)
				Expect(err).NotTo(HaveOccurred())
				a, err := spec.FromJson(data)
				Expect(err).NotTo(HaveOccurred())

				return proto.Equal(s, a)
			}

			Expect(quick.Check(fn, nil)).To(Succeed())
		})
	})

	Describe("Protobuf", func() {
		It("should round-trip", func() {
			fn := func(name string) bool {
				s := &tdlv1alpha1.Spec{Name: name}

				data, err := spec.ToProto(s)
				Expect(err).NotTo(HaveOccurred())
				a, err := spec.FromProto(data)
				Expect(err).NotTo(HaveOccurred())

				return proto.Equal(s, a)
			}

			Expect(quick.Check(fn, nil)).To(Succeed())
		})
	})

	Describe("Yaml", func() {
		It("should round-trip", func() {
			fn := func(name string) bool {
				s := &tdlv1alpha1.Spec{Name: name}

				data, err := spec.ToYaml(s)
				Expect(err).NotTo(HaveOccurred())
				a, err := spec.FromYaml(data)
				Expect(err).NotTo(HaveOccurred())

				return proto.Equal(s, a)
			}

			Expect(quick.Check(fn, nil)).To(Succeed())
		})
	})

	DescribeTable("MediaType",
		func(m tdl.MediaType) {
			fn := func(name string) bool {
				s := &tdlv1alpha1.Spec{Name: name}

				data, err := spec.ToMediaType(m, s)
				Expect(err).NotTo(HaveOccurred())
				a, err := spec.FromMediaType(m, data)
				Expect(err).NotTo(HaveOccurred())

				return proto.Equal(s, a)
			}

			Expect(quick.Check(fn, nil)).To(Succeed())
		},
		testing.MediaTypeEntries(),
	)
})
