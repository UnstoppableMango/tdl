package github_test

import (
	"context"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/unstoppablemango/tdl/pkg/plugin/github"
)

var _ = Describe("Github", func() {
	Describe("Prepare", Label("E2E"), func() {
		var workdir string

		BeforeEach(func() {
			tmp, err := os.MkdirTemp("", "")
			Expect(err).NotTo(HaveOccurred())
			Expect(os.Setenv("XDG_CACHE_HOME", tmp)).To(Succeed())
			workdir = tmp
		})

		AfterEach(func() {
			Expect(os.Unsetenv("XDG_CACHE_HOME")).To(Succeed())
		})

		It("should cache tdl-linux-amd64.tar.gz", func(ctx context.Context) {
			release := github.NewRelease("tdl-linux-amd64.tar.gz", "0.0.29")

			err := release.Prepare(ctx)

			Expect(err).NotTo(HaveOccurred())
			Expect(filepath.Join(workdir, "tdl-linux-amd64.tar.gz")).To(BeARegularFile())
		})

		It("should cache uml2ts", func(ctx context.Context) {
			release := github.NewRelease("tdl-linux-amd64.tar.gz", "0.0.29",
				github.WithArchiveContents("uml2ts"),
			)

			err := release.Prepare(ctx)

			Expect(err).NotTo(HaveOccurred())
		})

		It("should NOT cache unspecified artifacts", func(ctx context.Context) {
			release := github.NewRelease("tdl-linux-amd64.tar.gz", "0.0.29",
				github.WithArchiveContents("uml2ts"),
			)

			err := release.Prepare(ctx)

			Expect(err).NotTo(HaveOccurred())
		})
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
