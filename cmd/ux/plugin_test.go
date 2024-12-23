package main_test

import (
	"context"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("ux plugin", Label("E2E"), func() {
	Describe("pull", Ordered, func() {
		var (
			cachePath string
			binPath   string
		)

		BeforeAll(func() {
			var err error
			By("creating temp dirs for cache and bins")
			cachePath, err = os.MkdirTemp("", "")
			Expect(err).NotTo(HaveOccurred())
			binPath, err = os.MkdirTemp("", "")
			Expect(err).NotTo(HaveOccurred())

			By("setting the XDG env vars")
			Expect(os.Setenv("XDG_CACHE_HOME", cachePath)).To(Succeed())
			Expect(os.Setenv("XDG_BIN_HOME", binPath)).To(Succeed())
			Expect(os.Setenv("DISABLE_TUI", "true")).To(Succeed())
			Expect(os.Setenv("UX_LOG_LEVEL", "Debug")).To(Succeed())
		})

		AfterAll(func() {
			By("cleaning up cache and bin")
			Expect(os.Setenv("UX_LOG_LEVEL", "Error")).To(Succeed())
			Expect(os.RemoveAll(cachePath)).To(Succeed())
			Expect(os.RemoveAll(binPath)).To(Succeed())
		})

		It("should pull the plugin and cache file files", func(ctx context.Context) {
			cmd := UxCommand(ctx, "plugin", "pull",
				"https://github.com/UnstoppableMango/tdl/releases/tag/v0.0.32/tdl-linux-amd64.tar.gz",
			)

			ses, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)

			Expect(err).NotTo(HaveOccurred())
			Eventually(filepath.Join).WithArguments(cachePath, "ux", "tdl-linux-amd64.tar.gz").Should(BeARegularFile())
			Eventually(filepath.Join).WithArguments(binPath, "uml2ts").Should(BeARegularFile())
			Eventually(filepath.Join).WithArguments(binPath, "ux").Should(BeARegularFile())
			Eventually(ses.Out).Should(gbytes.Say("already cached"))
			Eventually(ses.Out).Should(gbytes.Say("Done\n"))
			Eventually(ses).Should(gexec.Exit(0))
		})

		It("should not pull the plugin once cached", func(ctx context.Context) {
			cmd := UxCommand(ctx, "plugin", "pull",
				"https://github.com/UnstoppableMango/tdl/releases/tag/v0.0.32/tdl-linux-amd64.tar.gz",
			)

			ses, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)

			Expect(err).NotTo(HaveOccurred())
			Eventually(filepath.Join).WithArguments(cachePath, "ux", "tdl-linux-amd64.tar.gz").Should(BeARegularFile())
			Eventually(filepath.Join).WithArguments(binPath, "uml2ts").Should(BeARegularFile())
			Eventually(filepath.Join).WithArguments(binPath, "ux").Should(BeARegularFile())
			Eventually(ses.Out).Should(gbytes.Say("bin exists: uml2ts"))
			Eventually(ses.Out).Should(gbytes.Say("bin exists: ux"))
			Eventually(ses.Out).Should(gbytes.Say("Done\n"))
			Eventually(ses).Should(gexec.Exit(0))
		})

		It("should re-pull the archive if it has been removed", func(ctx context.Context) {
			Expect(os.Remove(filepath.Join(cachePath, "ux", "tdl-linux-amd64.tar.gz"))).To(Succeed())
			cmd := UxCommand(ctx, "plugin", "pull",
				"https://github.com/UnstoppableMango/tdl/releases/tag/v0.0.32/tdl-linux-amd64.tar.gz",
			)

			ses, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)

			Expect(err).NotTo(HaveOccurred())
			Eventually(filepath.Join).WithArguments(cachePath, "ux", "tdl-linux-amd64.tar.gz").Should(BeARegularFile())
			Eventually(filepath.Join).WithArguments(binPath, "uml2ts").Should(BeARegularFile())
			Eventually(filepath.Join).WithArguments(binPath, "ux").Should(BeARegularFile())
			Eventually(ses.Out).Should(gbytes.Say("bin exists: uml2ts"))
			Eventually(ses.Out).Should(gbytes.Say("bin exists: ux"))
			Eventually(ses.Out).Should(gbytes.Say("Done\n"))
			Eventually(ses).Should(gexec.Exit(0))
		})

		It("should re-extract bins if they have been removed", func(ctx context.Context) {
			Expect(os.Remove(filepath.Join(binPath, "uml2ts"))).To(Succeed())
			cmd := UxCommand(ctx, "plugin", "pull",
				"https://github.com/UnstoppableMango/tdl/releases/tag/v0.0.32/tdl-linux-amd64.tar.gz",
			)

			ses, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)

			Expect(err).NotTo(HaveOccurred())
			Eventually(filepath.Join).WithArguments(cachePath, "ux", "tdl-linux-amd64.tar.gz").Should(BeARegularFile())
			Eventually(filepath.Join).WithArguments(binPath, "uml2ts").Should(BeARegularFile())
			Eventually(filepath.Join).WithArguments(binPath, "ux").Should(BeARegularFile())
			Eventually(ses.Out).Should(gbytes.Say("bin exists: ux"))
			Eventually(ses.Out).Should(gbytes.Say("Done\n"))
			Eventually(ses).Should(gexec.Exit(0))
		})
	})
})
