package main_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("ux plugin", Label("E2E"), func() {
	Describe("pull", Ordered, func() {
		const tdlUrl = "https://github.com/UnstoppableMango/tdl/releases/tag/v0.0.32/tdl-linux-amd64.tar.gz"

		var (
			cachePath string
			binPath   string
			envVars   []string
		)

		BeforeAll(func() {
			By("creating temp dirs for cache and bins")
			cachePath = GinkgoT().TempDir()
			binPath = GinkgoT().TempDir()

			By("creating env vars")
			envVars = []string{
				fmt.Sprintf("XDG_CACHE_HOME=%s", cachePath),
				fmt.Sprintf("XDG_BIN_HOME=%s", binPath),
				fmt.Sprintf("DISABLE_TUI=%s", "true"),
				fmt.Sprintf("UX_LOG_LEVEL=%s", "Debug"),
			}
		})

		AfterAll(func() {
			By("cleaning up env vars")
			Expect(os.Setenv("UX_LOG_LEVEL", "Error")).To(Succeed())
		})

		It("should pull the plugin and cache file files", func(ctx context.Context) {
			cmd := UxCommand(ctx, "plugin", "pull", tdlUrl)
			cmd.Env = append(cmd.Env, envVars...)

			ses, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)

			Expect(err).NotTo(HaveOccurred())
			Eventually(ses.Out, "15s").Should(gbytes.Say(`Done\n`))
			Eventually(ses).Should(gexec.Exit(0))
			Expect(filepath.Join(cachePath, "ux", "tdl-linux-amd64.tar.gz")).To(BeARegularFile())
			Expect(filepath.Join(binPath, "uml2ts")).To(BeARegularFile())
			Expect(filepath.Join(binPath, "ux")).To(BeARegularFile())
		})

		It("should not pull the plugin once cached", func(ctx context.Context) {
			cmd := UxCommand(ctx, "plugin", "pull", tdlUrl)
			cmd.Env = append(cmd.Env, envVars...)

			ses, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)

			Expect(err).NotTo(HaveOccurred())
			Eventually(ses.Out, "15s").Should(gbytes.Say(`Done\n`))
			Eventually(ses).Should(gexec.Exit(0))
			Expect(filepath.Join(cachePath, "ux", "tdl-linux-amd64.tar.gz")).To(BeARegularFile())
			Expect(filepath.Join(binPath, "uml2ts")).To(BeARegularFile())
			Expect(filepath.Join(binPath, "ux")).To(BeARegularFile())
		})

		It("should re-pull the archive if it has been removed", func(ctx context.Context) {
			Expect(os.Remove(filepath.Join(cachePath, "ux", "tdl-linux-amd64.tar.gz"))).To(Succeed())
			cmd := UxCommand(ctx, "plugin", "pull", tdlUrl)
			cmd.Env = append(cmd.Env, envVars...)

			ses, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)

			Expect(err).NotTo(HaveOccurred())
			Eventually(ses.Out, "15s").Should(gbytes.Say("Done\n"))
			Eventually(ses).Should(gexec.Exit(0))
			Expect(filepath.Join(cachePath, "ux", "tdl-linux-amd64.tar.gz")).To(BeARegularFile())
			Expect(filepath.Join(binPath, "uml2ts")).To(BeARegularFile())
			Expect(filepath.Join(binPath, "ux")).To(BeARegularFile())
		})

		It("should re-extract bins if they have been removed", func(ctx context.Context) {
			Expect(os.Remove(filepath.Join(binPath, "uml2ts"))).To(Succeed())
			cmd := UxCommand(ctx, "plugin", "pull", tdlUrl)
			cmd.Env = append(cmd.Env, envVars...)

			ses, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)

			Expect(err).NotTo(HaveOccurred())
			Eventually(ses.Out, "15s").Should(gbytes.Say("Done\n"))
			Eventually(ses).Should(gexec.Exit(0))
			Expect(filepath.Join(cachePath, "ux", "tdl-linux-amd64.tar.gz")).To(BeARegularFile())
			Expect(filepath.Join(binPath, "uml2ts")).To(BeARegularFile())
			Expect(filepath.Join(binPath, "ux")).To(BeARegularFile())
		})
	})
})
