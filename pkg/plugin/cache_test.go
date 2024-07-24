package plugin_test

import (
	"log/slog"
	"os"

	"github.com/google/go-github/v63/github"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/unstoppablemango/tdl/pkg/plugin"
)

var _ = Describe("PluginCache", func() {
	var cacheDir string

	BeforeEach(func() {
		// if _, found := os.LookupEnv("CI"); !found {
		// 	Skip("Only running cache test in CI")
		// }

		By("creating a temporary directory")
		tmp, err := os.MkdirTemp("", "plugin-cache-test")
		Expect(err).NotTo(HaveOccurred())
		cacheDir = tmp
	})

	AfterEach(func() {
		err := os.RemoveAll(cacheDir)
		Expect(err).NotTo(HaveOccurred())
	})

	DescribeTable("should retrieve path",
		Entry("uml2go", "uml2go"),
		Entry("uml2pcl", "uml2pcl"),
		func(name string) {
			gh := github.NewClient(nil)
			cache := plugin.NewCache(gh, cacheDir, slog.Default())

			result, err := cache.PathFor(name)

			Expect(err).NotTo(HaveOccurred())
			Expect(result).NotTo(BeEmpty())
		},
	)
})
