package gen_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/unstoppablemango/tdl/pkg/gen"
	"github.com/unstoppablemango/tdl/pkg/tdl"
	tdlv1alpha1 "github.com/unstoppablemango/tdl/pkg/unmango/dev/tdl/v1alpha1"
)

var _ = Describe("Generator", func() {
	Describe("Lift", func() {
		It("should invoke the generator function", func() {
			sentinel := false
			var fn gen.Func = func(*tdlv1alpha1.Spec, tdl.Sink) error {
				sentinel = true
				return nil
			}

			gen := gen.Lift(fn)

			Expect(gen.Execute(nil, nil)).To(Succeed())
			Expect(sentinel).To(BeTrueBecause("the generator function is invoked"))
		})
	})

	Describe("New", func() {
		It("should invoke the generator function", func() {
			sentinel := false
			var fn gen.Func = func(*tdlv1alpha1.Spec, tdl.Sink) error {
				sentinel = true
				return nil
			}

			gen := gen.New(fn)

			Expect(gen.Execute(nil, nil)).To(Succeed())
			Expect(sentinel).To(BeTrueBecause("the generator function is invoked"))
		})
	})
})
