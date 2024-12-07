package github_test

import (
	"context"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/afero"

	"github.com/unstoppablemango/tdl/pkg/plugin/github"
	"github.com/unstoppablemango/tdl/pkg/testing"
)

var _ = Describe("Github", func() {
	var client github.Client
	var cache *testing.Cache

	BeforeEach(func() {
		client = github.NewClient(testing.NewGitHubClient())
		cache = testing.NewCache(afero.NewOsFs())
	})

	It("should cache tdl-linux-amd64.tar.gz", Label("E2E"), func(ctx context.Context) {
		release := github.NewRelease("tdl-linux-amd64.tar.gz", "0.0.29",
			github.WithClient(client),
		)

		err := release.Cache(ctx, cache)

		Expect(err).NotTo(HaveOccurred())
		dir := cache.Dir()
		path := filepath.Join(dir, "tdl-linux-amd64.tar.gz")
		Expect(path).To(BeAnExistingFile())
	})

	It("should cache uml2ts", Label("E2E"), func(ctx context.Context) {
		release := github.NewRelease("tdl-linux-amd64.tar.gz", "0.0.29",
			github.WithClient(client),
			github.WithArchiveContents("uml2ts"),
		)

		err := release.Cache(ctx, cache)

		Expect(err).NotTo(HaveOccurred())
		dir := cache.Dir()
		path := filepath.Join(dir, "uml2ts")
		Expect(path).To(BeARegularFile())
		Expect(release.Cached(cache)).To(BeTrueBecause("The bin was cached"))
	})

	It("should NOT cache unspecified artifacts", Label("E2E"), func(ctx context.Context) {
		release := github.NewRelease("tdl-linux-amd64.tar.gz", "0.0.29",
			github.WithClient(client),
			github.WithArchiveContents("uml2ts"),
		)

		err := release.Cache(ctx, cache)

		Expect(err).NotTo(HaveOccurred())
		dir := cache.Dir()
		path := filepath.Join(dir, "uml2go")
		Expect(path).NotTo(BeARegularFile())
	})

	Describe("String", func() {
		It("should return the url of the github release", func() {
			release := github.NewRelease("tdl-linux-amd64.tar.gz", "0.0.29",
				github.WithClient(client),
				github.WithArchiveContents("uml2ts"),
			)

			result := release.String()

			Expect(result).To(Equal("https://github.com/UnstoppableMango/tdl/releases/download/v0.0.29/tdl-linux-amd64.tar.gz"))
		})
	})
})
