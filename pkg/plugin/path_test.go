package plugin_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/unstoppablemango/tdl/pkg/gen"
	"github.com/unstoppablemango/tdl/pkg/plugin"
)

var _ = Describe("Path", func() {
	It("should use a default priority", func() {
		p := plugin.FromPath("thing")

		o, ok := p.(plugin.Ordered)
		Expect(ok).To(BeTrueBecause("the fromPath plugin has an order"))
		Expect(o.Order()).To(Equal(69))
	})

	It("should stringify", func() {
		p := plugin.FromPath("path-thing")

		Expect(p.String()).To(Equal("PATH: path-thing"))
	})

	Describe("Generator", func() {
		It("should look up plugin from path", func() {
			p := plugin.FromPath("uml2ts")

			g, err := p.Generator(nil)

			Expect(err).NotTo(HaveOccurred())
			Expect(g).To(BeAssignableToTypeOf(&gen.Cli{}))
		})
	})
})