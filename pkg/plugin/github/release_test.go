package github_test

import (
	"context"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/unstoppablemango/tdl/pkg/plugin/github"
	"github.com/unstoppablemango/tdl/pkg/testing"
)

var _ = Describe("Github", Label("E2E"), func() {
	var client github.Client
	var cache *testing.Cache

	BeforeEach(func() {
		client = github.NewClient(testing.NewGitHubClient())
		cache = testing.NewCache(nil)
	})

	It("should cache tdl-linux-amd64.tar.gz", func(ctx context.Context) {
		release := github.NewRelease("tdl-linux-amd64.tar.gz", "0.0.29",
			github.WithClient(client),
			github.WithCache(cache),
		)

		err := release.Cache(ctx)

		Expect(err).NotTo(HaveOccurred())
		dir, err := cache.Dir()
		Expect(err).NotTo(HaveOccurred())
		path := filepath.Join(dir, "tdl-linux-amd64.tar.gz")
		Expect(path).To(BeAnExistingFile())
	})
})
