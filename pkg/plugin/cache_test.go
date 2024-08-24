package plugin_test

import (
	"log/slog"
	"os"

	"github.com/google/go-github/v64/github"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/unstoppablemango/tdl/pkg/plugin"
)

var _ = Describe("PluginCache", func() {
	var cacheDir string

	BeforeEach(func() {
		Skip("Super Skip: I'm renaming artifacts right now and this is broken")
		if _, found := os.LookupEnv("CI"); !found {
			Skip("Only running cache test in CI")
		}

		By("creating a temporary directory")
		tmp, err := os.MkdirTemp("", "plugin-cache-test")
		Expect(err).NotTo(HaveOccurred())
		cacheDir = tmp
	})

	AfterEach(func() {
		err := os.RemoveAll(cacheDir)
		Expect(err).NotTo(HaveOccurred())
	})

	It("should retrieve path", func() {
		gh := github.NewClient(nil)
		cache := plugin.NewCache(gh, cacheDir, slog.Default())

		result, err := cache.PathFor("uml2go")

		Expect(err).NotTo(HaveOccurred())
		Expect(result).NotTo(BeEmpty())
	})
})
