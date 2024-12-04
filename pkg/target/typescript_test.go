package target_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/unmango/go/iter"
	tdl "github.com/unstoppablemango/tdl/pkg"
	"github.com/unstoppablemango/tdl/pkg/plugin"
	"github.com/unstoppablemango/tdl/pkg/target"
	"github.com/unstoppablemango/tdl/pkg/testing"
)

var _ = Describe("Typescript", func() {
	Describe("Generator", func() {
		It("should choose uml2ts", func() {
			chosen, err := target.TypeScript.Choose(
				iter.Singleton(plugin.Uml2Ts),
			)

			Expect(err).NotTo(HaveOccurred())
			Expect(chosen).NotTo(BeNil()) // TODO
		})

		It("should ignore unsupported generators", func() {
			g := (&testing.MockPlugin{}).WithString(func() string {
				return "test"
			})

			_, err := target.TypeScript.Choose(
				iter.Singleton[tdl.Plugin](g),
			)

			Expect(err).To(MatchError("no suitable plugin"))
		})
	})
})
