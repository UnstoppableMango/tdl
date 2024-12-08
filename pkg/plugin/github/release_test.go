package github_test

import (
	"context"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/afero"

	"github.com/unstoppablemango/tdl/pkg/meta"
	"github.com/unstoppablemango/tdl/pkg/plugin/github"
	"github.com/unstoppablemango/tdl/pkg/testing"
)

var _ = Describe("Github", func() {
	var cache *testing.DirCache

	BeforeEach(func() {
		cache = testing.NewCache(afero.NewOsFs())
	})

	It("should cache tdl-linux-amd64.tar.gz", Label("E2E"), func(ctx context.Context) {
		release := github.NewRelease("tdl-linux-amd64.tar.gz", "0.0.29")

		_, err := release.Generator(ctx, meta.Empty())

		Expect(err).NotTo(HaveOccurred())
		dir := cache.Dir()
		path := filepath.Join(dir, "tdl-linux-amd64.tar.gz")
		Expect(path).To(BeAnExistingFile())
	})

	It("should cache uml2ts", Label("E2E"), func(ctx context.Context) {
		release := github.NewRelease("tdl-linux-amd64.tar.gz", "0.0.29",
			github.WithArchiveContents("uml2ts"),
		)

		_, err := release.Generator(ctx, meta.Empty())

		Expect(err).NotTo(HaveOccurred())
		dir := cache.Dir()
		path := filepath.Join(dir, "uml2ts")
		Expect(path).To(BeARegularFile())
		// Expect(release.Cached(cache)).To(BeTrueBecause("The bin was cached"))
	})

	It("should NOT cache unspecified artifacts", Label("E2E"), func(ctx context.Context) {
		release := github.NewRelease("tdl-linux-amd64.tar.gz", "0.0.29",
			github.WithArchiveContents("uml2ts"),
		)

		_, err := release.Generator(ctx, meta.Empty())

		Expect(err).NotTo(HaveOccurred())
		dir := cache.Dir()
		path := filepath.Join(dir, "uml2go")
		Expect(path).NotTo(BeARegularFile())
	})

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
