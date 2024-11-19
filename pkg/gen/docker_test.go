package gen_test

import (
	"context"

	"github.com/docker/docker/client"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/unstoppablemango/tdl/pkg/gen"
	. "github.com/unstoppablemango/tdl/pkg/testing/matcher"
	tdlv1alpha1 "github.com/unstoppablemango/tdl/pkg/unmango/dev/tdl/v1alpha1"
)

var _ = Describe("Docker", func() {
	It("should work", func(ctx context.Context) {
		g := gen.NewDocker("ghcr.io/unstoppablemango/uml2ts:latest",
			client.WithAPIVersionNegotiation(),
		)
		spec := &tdlv1alpha1.Spec{}

		fs, err := g.Execute(ctx, spec)

		Expect(err).NotTo(HaveOccurred())
		Expect(fs).To(ContainFile("out"))
	})
})
