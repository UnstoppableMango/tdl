package plugin_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	tdl "github.com/unstoppablemango/tdl/pkg"
	"github.com/unstoppablemango/tdl/pkg/plugin"
	"github.com/unstoppablemango/tdl/pkg/testing"
)

var _ = Describe("Aggregate", func() {
	It("should consist of the given plugins", func() {
		p := testing.NewMockPlugin()

		result := plugin.NewAggregate(p)

		Expect(result).To(ConsistOf(p))
	})

	It("should pick the given generator", func() {
		g := testing.NewMockGenerator()
		p := testing.NewMockPlugin().
			WithGenerator(func(tdl.Target) (tdl.Generator, error) {
				return g, nil
			})
		agg := plugin.NewAggregate(p)

		result, err := agg.Generator(testing.NewMockTarget())

		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(BeIdenticalTo(g))
	})
})
