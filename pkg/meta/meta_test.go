package meta_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/unstoppablemango/tdl/pkg/meta"
)

var _ = Describe("Meta", func() {
	Describe("Supports", func() {
		It("should match when over half of the keys match", func() {
			a := &meta.Map{"name": "blue", "lang": "ts", "desc": "foo"}
			b := &meta.Map{"name": "blue", "lang": "ts", "desc": "bar"}

			s := meta.Supports(a, b)

			Expect(s).To(BeTrueBecause("it works"))
		})

		It("should not match when less than half of the keys match", func() {
			a := &meta.Map{"name": "blue", "lang": "go", "desc": "foo"}
			b := &meta.Map{"name": "blue", "lang": "ts", "desc": "bar"}

			s := meta.Supports(a, b)

			Expect(s).To(BeFalseBecause("it works"))
		})
	})
})
