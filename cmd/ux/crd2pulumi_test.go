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

var _ = Describe("tool crd2pulumi", func() {
	var (
		crdPath string
		out     string
	)

	BeforeEach(func() {
		wd, err := os.Getwd()
		Expect(err).NotTo(HaveOccurred())
		crdPath = filepath.Join(wd, "testdata", "crd.yaml")
		out = GinkgoT().TempDir()
	})

	It("should print nodejs paths", func(ctx context.Context) {
		cmd := UxCommand(ctx, "tool", "crd2pulumi",
			crdPath, "--", "--nodejs",
		)
		ses, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
		Expect(err).NotTo(HaveOccurred())

		Eventually(ses.Out, "5s").Should(gbytes.Say("nodejs/index.ts"))
		Eventually(ses.Out).Should(gbytes.Say("nodejs/types/index.ts"))
		Eventually(ses.Out).Should(gbytes.Say("nodejs/types/input.ts"))
		Eventually(ses.Out).Should(gbytes.Say("nodejs/types/output.ts"))

		Eventually(ses).Should(gexec.Exit(0))
	})

	It("should generate nodejs files", func(ctx context.Context) {
		cmd := UxCommand(ctx, "tool", "crd2pulumi",
			crdPath, "--output", out, "--", "--nodejs",
		)
		ses, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
		Expect(err).NotTo(HaveOccurred())

		Eventually(ses).Should(gexec.Exit(0))
		Expect(filepath.Join(out, "nodejs", "index.ts")).To(BeARegularFile())
		Expect(filepath.Join(out, "nodejs", "types", "index.ts")).To(BeARegularFile())
		Expect(filepath.Join(out, "nodejs", "types", "input.ts")).To(BeARegularFile())
		Expect(filepath.Join(out, "nodejs", "types", "output.ts")).To(BeARegularFile())
	})
})
