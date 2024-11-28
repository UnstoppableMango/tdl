package docker_test

import (
	"context"

	"github.com/docker/docker/client"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/unstoppablemango/tdl/pkg/gen/docker"
	. "github.com/unstoppablemango/tdl/pkg/testing/matcher"
	tdlv1alpha1 "github.com/unstoppablemango/tdl/pkg/unmango/dev/tdl/v1alpha1"
)

var _ = Describe("Docker", func() {
	It("should work", func(ctx context.Context) {
		client, err := client.NewClientWithOpts(client.WithAPIVersionNegotiation())
		Expect(err).NotTo(HaveOccurred())
		g := docker.New(client, "ghcr.io/unstoppablemango/uml2ts:v0.0.30")
		spec := &tdlv1alpha1.Spec{}

		fs, err := g.Execute(ctx, spec)

		Expect(err).NotTo(HaveOccurred())
		Expect(fs).To(ContainFile("stdout"))
	})
})
