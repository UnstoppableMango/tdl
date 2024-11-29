package docker_test

import (
	"context"
	"io"

	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/unstoppablemango/tdl/pkg/gen/docker"
	. "github.com/unstoppablemango/tdl/pkg/testing/matcher"
	tdlv1alpha1 "github.com/unstoppablemango/tdl/pkg/unmango/dev/tdl/v1alpha1"
)

var _ = Describe("Docker", func() {
	var testClient client.APIClient

	BeforeEach(func(ctx context.Context) {
		var err error
		testClient, err = client.NewClientWithOpts(
			client.WithAPIVersionNegotiation(),
		)
		Expect(err).NotTo(HaveOccurred())
	})

	When("the image exists", Label("E2E"), func() {
		BeforeEach(func(ctx context.Context) {
			reader, err := testClient.ImagePull(ctx,
				"ghcr.io/unstoppablemango/uml2ts:v0.0.30",
				image.PullOptions{},
			)
			Expect(err).NotTo(HaveOccurred())
			_, err = io.ReadAll(reader)
			Expect(err).NotTo(HaveOccurred())
			Expect(reader.Close()).To(Succeed())
		})

		It("should work", func(ctx context.Context) {
			g := docker.New(testClient, "ghcr.io/unstoppablemango/uml2ts:v0.0.30")
			spec := &tdlv1alpha1.Spec{}

			fs, err := g.Execute(ctx, spec)

			Expect(err).NotTo(HaveOccurred())
			Expect(fs).To(ContainFile("stdout"))
		})
	})
})
