package plugin_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/unstoppablemango/tdl/pkg/plugin"
	"github.com/unstoppablemango/tdl/pkg/plugin/github"
	"github.com/unstoppablemango/tdl/pkg/testing"
)

var _ = Describe("Github", func() {
	var release plugin.GitHubRelease

	BeforeEach(func() {
		github := github.NewClient(testing.NewGitHubClient())
		release = plugin.NewGitHubRelease(github, "uml2ts", "0.0.29")
	})

	It("should cache uml2ts", func(ctx context.Context) {
		err := release.Cache(ctx)

		Expect(err).NotTo(HaveOccurred())
	})
})
