package github_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/unstoppablemango/tdl/pkg/plugin/github"
)

var _ = Describe("Github", Pending, func() {
	// var cache *testing.Cache

	// BeforeEach(func() {
	// 	cache, _ = testing.NewTmpCache(GinkgoT())
	// })

	// It("should cache tdl-linux-amd64.tar.gz", Label("E2E"), func(ctx context.Context) {
	// 	release := github.NewRelease("tdl-linux-amd64.tar.gz", "0.0.29")

	// 	_, err := release.Generator(ctx, meta.Empty())

	// 	Expect(err).NotTo(HaveOccurred())
	// 	Expect(cache.Get("tdl-linux-amd64.tar.gz")).NotTo(BeNil())
	// })

	// It("should cache uml2ts", Label("E2E"), func(ctx context.Context) {
	// 	release := github.NewRelease("tdl-linux-amd64.tar.gz", "0.0.29",
	// 		github.WithArchiveContents("uml2ts"),
	// 	)

	// 	_, err := release.Generator(ctx, meta.Empty())

	// 	Expect(err).NotTo(HaveOccurred())
	// 	Expect(cache.Get("uml2ts")).NotTo(BeNil())
	// })

	// It("should NOT cache unspecified artifacts", Label("E2E"), func(ctx context.Context) {
	// 	release := github.NewRelease("tdl-linux-amd64.tar.gz", "0.0.29",
	// 		github.WithArchiveContents("uml2ts"),
	// 	)

	// 	_, err := release.Generator(ctx, meta.Empty())

	// 	Expect(err).NotTo(HaveOccurred())
	// 	Expect(cache.Get("uml2go")).NotTo(BeNil())
	// })

	Describe("String", func() {
		It("should return the url of the github release", func() {
			release := github.NewRelease("tdl-linux-amd64.tar.gz", "0.0.29",
				github.WithArchiveContents("uml2ts"),
			)

			result := release.String()

			Expect(result).To(Equal("https://github.com/UnstoppableMango/tdl/releases/download/v0.0.29/tdl-linux-amd64.tar.gz"))
		})
	})
})
