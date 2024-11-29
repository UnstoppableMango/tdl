package target_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/unstoppablemango/tdl/pkg/plugin"
	"github.com/unstoppablemango/tdl/pkg/target"
)

var _ = Describe("Pulumi", func() {
	It("should have a name", func() {
		name := target.Pulumi.String()

		Expect(name).To(Equal("Pulumi"))
	})

	It("should select crd2pulumi", func() {
		g, err := target.Pulumi.Generator(plugin.Static())

		Expect(err).NotTo(HaveOccurred())
		Expect(g.String()).To(Equal("crd2pulumi"))
	})
})
