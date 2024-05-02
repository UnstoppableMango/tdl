package pcl

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pulumi/pulumi/pkg/v3/codegen/schema"
)

var _ = Describe("Convert", func() {
	var converter PclConverter

	BeforeEach(func() {
		converter = Converter
	})

	It("should convert from pcl", func() {
		pcl := schema.PackageSpec{
			Name:        "test-name",
			Description: "test-description",
			DisplayName: "Test Name",
			Repository:  "github.com/UnstoppableMango/tdl",
			Keywords:    []string{"pulumi", "thing"},
			Version:     "v0.1.0",
		}

		spec, err := converter.FromPcl(context.Background(), pcl)
		Expect(err).To(BeNil())
		Expect(spec).NotTo(BeNil())

		Expect(spec.Name).NotTo(BeEmpty())
		Expect(spec.Name).To(Equal(pcl.Name))

		Expect(spec.Description).NotTo(BeEmpty())
		Expect(spec.Description).To(Equal(pcl.Description))

		Expect(spec.DisplayName).NotTo(BeEmpty())
		Expect(spec.DisplayName).To(Equal(pcl.DisplayName))

		Expect(spec.Source).NotTo(BeEmpty())
		Expect(spec.Source).To(Equal(pcl.Repository))

		Expect(spec.Labels).NotTo(BeEmpty())
		Expect(spec.Labels["keywords"]).To(BeEquivalentTo("pulumi,thing"))

		Expect(spec.Version).NotTo(BeEmpty())
		Expect(spec.Version).To(Equal(pcl.Version))
	})
})
