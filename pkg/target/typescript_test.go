package target_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/unmango/go/slices"
	tdl "github.com/unstoppablemango/tdl/pkg"
	"github.com/unstoppablemango/tdl/pkg/plugin"
	"github.com/unstoppablemango/tdl/pkg/target"
	"github.com/unstoppablemango/tdl/pkg/testing"
)

var _ = Describe("Typescript", func() {
	It("should list the uml2ts plugin", func() {
		expected := plugin.NewAggregate(plugin.Uml2Ts)

		plugins := target.TypeScript.Plugins()

		Expect(slices.Collect(plugins)).To(ConsistOf(expected))
	})

	Describe("Choose", func() {
		It("should choose uml2ts", func() {
			expected, err := plugin.Uml2Ts.SinkGenerator(target.TypeScript)
			Expect(err).NotTo(HaveOccurred())

			chosen, err := target.TypeScript.Choose([]tdl.SinkGenerator{expected})

			Expect(err).NotTo(HaveOccurred())
			Expect(chosen).To(BeIdenticalTo(expected))
		})

		It("should ignore unsupported generators", func() {
			g := testing.NewMockGenerator()

			_, err := target.TypeScript.Choose([]tdl.SinkGenerator{g})

			Expect(err).To(MatchError(ContainSubstring("not a CLI")))
		})
	})
})
