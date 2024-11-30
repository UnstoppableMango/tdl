package plugin_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

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

		Expect(p.String()).To(Equal("path-thing"))
	})

	Describe("Generator", func() {
		It("should look up plugin from path", func(ctx context.Context) {
			p := plugin.FromPath("uml2ts")

			g, err := p.Generator(ctx, nil)

			Expect(err).NotTo(HaveOccurred())
			Expect(g.String()).To(ContainSubstring("uml2ts"))
		})
	})
})
