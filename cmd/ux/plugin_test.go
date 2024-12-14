package main_test

import (
	"context"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ux plugin", func() {
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

			out, err := cmd.CombinedOutput()

			Expect(err).NotTo(HaveOccurred())
			Expect(filepath.Join(cachePath, "ux", "tdl-linux-amd64.tar.gz")).To(BeARegularFile(), string(out))
			Expect(filepath.Join(binPath, "uml2ts")).To(BeARegularFile(), string(out))
			Expect(filepath.Join(binPath, "ux")).To(BeARegularFile(), string(out))
			Expect(string(out)).NotTo(ContainSubstring("already cached"))
			Expect(string(out)).To(ContainSubstring("Done\n"))
		})

		It("should not pull the plugin once cached", func(ctx context.Context) {
			cmd := UxCommand(ctx, "plugin", "pull",
				"https://github.com/UnstoppableMango/tdl/releases/tag/v0.0.32/tdl-linux-amd64.tar.gz",
			)

			out, err := cmd.CombinedOutput()

			Expect(err).NotTo(HaveOccurred())
			Expect(filepath.Join(cachePath, "ux", "tdl-linux-amd64.tar.gz")).To(BeARegularFile(), string(out))
			Expect(filepath.Join(binPath, "uml2ts")).To(BeARegularFile(), string(out))
			Expect(filepath.Join(binPath, "ux")).To(BeARegularFile(), string(out))
			Expect(string(out)).To(ContainSubstring("bin exists: uml2ts"))
			Expect(string(out)).To(ContainSubstring("bin exists: ux"))
			Expect(string(out)).To(ContainSubstring("Done\n"))
		})
	})
})
