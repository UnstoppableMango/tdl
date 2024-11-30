package docker_test

import (
	"context"

	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/unstoppablemango/tdl/pkg/plugin/docker"
)

var _ = Describe("Plugin", Serial, func() {
	var testClient client.APIClient

	BeforeEach(func() {
		var err error
		By("creating a docker client")
		testClient, err = client.NewClientWithOpts(
			client.WithAPIVersionNegotiation(),
		)
		Expect(err).NotTo(HaveOccurred())
	})

	It("should create a new plugin", func() {
		p := docker.New(testClient, "blah")

		Expect(p).NotTo(BeNil())
	})

	It("should fetch a docker generator", func(ctx context.Context) {
		p := docker.New(testClient, "ghcr.io/unstoppablemango/uml2ts:v0.0.30")

		g, err := p.Generator(ctx, nil)

		Expect(err).NotTo(HaveOccurred())
		Expect(g.String()).To(Equal("ghcr.io/unstoppablemango/uml2ts:v0.0.30"))
	})

	When("the image does not exist", Label("E2E"), func() {
		BeforeEach(func(ctx context.Context) {
			exists, err := docker.ImageExists(ctx, testClient, "ghcr.io/unstoppablemango/uml2ts:v0.0.31")
			Expect(err).NotTo(HaveOccurred())
			if !exists {
				return
			}

			_, err = testClient.ImageRemove(ctx,
				"ghcr.io/unstoppablemango/uml2ts:v0.0.31",
				image.RemoveOptions{},
			)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should pull a fresh image", func(ctx context.Context) {
			p := docker.New(testClient, "ghcr.io/unstoppablemango/uml2ts:v0.0.31")

			g, err := p.Generator(ctx, nil)

			Expect(err).NotTo(HaveOccurred())
			Expect(g.String()).To(Equal("ghcr.io/unstoppablemango/uml2ts:v0.0.31"))
		})
	})
})
