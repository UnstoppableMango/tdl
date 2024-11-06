package plugin_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	tdl "github.com/unstoppablemango/tdl/pkg"
	"github.com/unstoppablemango/tdl/pkg/gen"
	"github.com/unstoppablemango/tdl/pkg/plugin"
	"github.com/unstoppablemango/tdl/pkg/testing"
)

var _ = Describe("Aggregate", func() {
	It("should consist of the given plugins", func() {
		result := plugin.NewAggregate(plugin.Uml2Ts)

		Expect(result).To(ConsistOf(plugin.Uml2Ts))
	})

	It("should pick the given generator", func() {
		t := &testing.MockTarget{}
		p := testing.NewMockPlugin().
			WithGenerator(func(t tdl.Target) (tdl.Generator, error) {
				return nil, nil
			})

		agg := plugin.NewAggregate(p)

		result, err := agg.Generator(t)

		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(BeAssignableToTypeOf(&gen.Cli{}))
	})
})
