package gen_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/afero"
	"github.com/unstoppablemango/tdl/pkg/gen"
	tdlv1alpha1 "github.com/unstoppablemango/tdl/pkg/unmango/dev/tdl/v1alpha1"
)

var _ = Describe("Generator", func() {
	Describe("Lift", func() {
		It("should invoke the generator function", func(ctx context.Context) {
			sentinel := false
			var fn gen.Func = func(context.Context, *tdlv1alpha1.Spec) (afero.Fs, error) {
				sentinel = true
				return nil, nil
			}

			gen := gen.Lift(fn)

			_, err := gen.Execute(ctx, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(sentinel).To(BeTrueBecause("the generator function is invoked"))
		})
	})

	Describe("New", func() {
		It("should invoke the generator function", func(ctx context.Context) {
			sentinel := false
			var fn gen.Func = func(context.Context, *tdlv1alpha1.Spec) (afero.Fs, error) {
				sentinel = true
				return nil, nil
			}

			gen := gen.New(fn)

			_, err := gen.Execute(ctx, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(sentinel).To(BeTrueBecause("the generator function is invoked"))
		})
	})
})
