package target_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	tdl "github.com/unstoppablemango/tdl/pkg"
	"github.com/unstoppablemango/tdl/pkg/meta"
	"github.com/unstoppablemango/tdl/pkg/target"
	"github.com/unstoppablemango/tdl/pkg/testing"
)

var _ = Describe("Tool", func() {
	It("should choose the named tool", func() {
		t := target.NewTool("thing")

		p, err := t.Choose(func(yield func(tdl.Plugin) bool) {
			yield(&testing.MockPlugin{
				MetaValue: meta.Map{
					"name": "thing",
				},
				SupportsFunc: func(t tdl.Target) bool {
					return true
				},
			})
		})

		Expect(err).NotTo(HaveOccurred())
		Expect(p).NotTo(BeNil())
	})

	It("should not match other tools", func() {
		t := target.NewTool("thing")

		_, err := t.Choose(func(yield func(tdl.Plugin) bool) {
			yield(&testing.MockPlugin{
				MetaValue: meta.Map{
					"name": "blah",
				},
				SupportsFunc: func(t tdl.Target) bool {
					return false
				},
			})
		})

		Expect(err).To(MatchError("no match for target: thing"))
	})
})
