package plugin_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/unstoppablemango/tdl/pkg/plugin"
	"github.com/unstoppablemango/tdl/pkg/testing"
)

var _ = Describe("Unwrap", func() {
	It("should unwrap an aggregate plugin", func() {
		p := testing.NewMockPlugin()
		agg := plugin.NewAggregate(p)

		r := plugin.Unwrap(agg)

		Expect(r).To(ConsistOf(p))
	})

	It("should unwarp multiple aggregate plugins", func() {
		p1 := testing.NewMockPlugin()
		p2 := testing.NewMockPlugin()
		agg := plugin.NewAggregate(p1, p2)

		r := plugin.Unwrap(agg)

		Expect(r).To(ConsistOf(p1, p2))
	})

	It("should unwarp all aggregate plugins", func() {
		p1 := testing.NewMockPlugin()
		p2 := testing.NewMockPlugin()
		agg := plugin.NewAggregate(p1, p2)

		r := plugin.UnwrapAll(agg)

		Expect(r).To(ConsistOf(p1, p2))
	})

	It("should unwrap nested aggregate plugin", func() {
		p := testing.NewMockPlugin()
		agg := plugin.NewAggregate(
			plugin.NewAggregate(p),
		)

		r := plugin.Unwrap(agg)

		Expect(r).To(ConsistOf(p))
	})

	It("should unwrap nested aggregate plugins up to max depth", func() {
		p := testing.NewMockPlugin()
		agg := plugin.NewAggregate(p)
		for i := 0; i < plugin.UnwrapDepth-1; i++ {
			agg = plugin.NewAggregate(agg)
		}

		r := plugin.Unwrap(agg)

		Expect(r).To(ConsistOf(p))
	})

	It("should stop unwrapping nested aggregate plugins after max depth", func() {
		p := testing.NewMockPlugin()
		agg := plugin.NewAggregate(p)
		for i := 0; i < plugin.UnwrapDepth; i++ {
			agg = plugin.NewAggregate(agg)
		}

		r := plugin.Unwrap(agg)

		Expect(r).To(ConsistOf(
			plugin.NewAggregate(p),
		))
	})
})
