package gen_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	tdl "github.com/unstoppablemango/tdl/pkg"
	"github.com/unstoppablemango/tdl/pkg/gen"
	tdlv1alpha1 "github.com/unstoppablemango/tdl/pkg/unmango/dev/tdl/v1alpha1"
)

var _ = Describe("Generator", func() {
	Describe("Lift", func() {
		It("should invoke the generator function", func(ctx context.Context) {
			sentinel := false
			var fn gen.Func = func(context.Context, *tdlv1alpha1.Spec, tdl.Sink) error {
				sentinel = true
				return nil
			}

			gen := gen.Lift(fn)

			Expect(gen.Execute(ctx, nil, nil)).To(Succeed())
			Expect(sentinel).To(BeTrueBecause("the generator function is invoked"))
		})
	})

	Describe("New", func() {
		It("should invoke the generator function", func(ctx context.Context) {
			sentinel := false
			var fn gen.Func = func(context.Context, *tdlv1alpha1.Spec, tdl.Sink) error {
				sentinel = true
				return nil
			}

			gen := gen.New(fn)

			Expect(gen.Execute(ctx, nil, nil)).To(Succeed())
			Expect(sentinel).To(BeTrueBecause("the generator function is invoked"))
		})
	})
})
