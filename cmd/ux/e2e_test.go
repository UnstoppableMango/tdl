package main_test

import (
	"context"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
	"github.com/spf13/afero"
	. "github.com/unmango/go/testing/matcher"

	"github.com/unstoppablemango/tdl/pkg/conform"
	"github.com/unstoppablemango/tdl/pkg/gen/cli"
)

var _ = Describe("End to end", Label("E2E"), func() {
	Describe("TypeScript Conformance", FlakeAttempts(5), func() {
		Describe("stdout", func() {
			generator := cli.New("ux",
				cli.WithArgs("gen", "ts", "-"),
				cli.ExpectStdout,
			)

			conform.DescribeGenerator(typescriptSuite, generator)
		})
	})

	It("should pass my excessive sanity check", func() {
		Expect(bin).NotTo(BeEmpty())
	})

	Describe("gen", func() {
		It("should error when input is not provided", func(ctx context.Context) {
			cmd := UxCommand(ctx, "gen", "ts")

			ses, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)

			Expect(err).NotTo(HaveOccurred())
			Eventually(ses.Err).Should(gbytes.Say("no input specified\n"))
			Eventually(ses).Should(gexec.Exit(1))
		})

		It("should read spec from yaml file", FlakeAttempts(5), func(ctx context.Context) {
			input := filepath.Join(tsSuitePath(), "interface", "source.yml")
			output, err := os.ReadFile(filepath.Join(tsSuitePath(), "interface", "target.ts"))
			Expect(err).NotTo(HaveOccurred())
			cmd := UxCommand(ctx, "gen", "ts", input)

			ses, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)

			Expect(err).NotTo(HaveOccurred())
			Eventually(ses.Out).Should(gbytes.Say(string(output)))
			Eventually(ses).Should(gexec.Exit(0))
		})

		It("should error when input does not exist", func(ctx context.Context) {
			input := filepath.Join("fkjdslfkdjlsf")
			cmd := UxCommand(ctx, "gen", "ts", input)

			ses, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)

			Expect(err).NotTo(HaveOccurred())
			Eventually(ses.Err).Should(gbytes.Say("parsing run config: stat fkjdslfkdjlsf: no such file or directory\n"))
			Eventually(ses).Should(gexec.Exit(1))
		})

		It("should write to output file", FlakeAttempts(5), func(ctx context.Context) {
			input := filepath.Join(tsSuitePath(), "interface", "source.yml")
			output := filepath.Join(GinkgoT().TempDir(), "index.ts")
			expected, err := os.ReadFile(filepath.Join(tsSuitePath(), "interface", "target.ts"))
			Expect(err).NotTo(HaveOccurred())
			cmd := UxCommand(ctx, "gen", "ts", input, output)

			ses, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)

			Expect(err).NotTo(HaveOccurred())
			Consistently(ses.Out).ShouldNot(gbytes.Say(".+"))
			Eventually(afero.NewOsFs).Should(ContainFileWithBytes(output, expected))
			Eventually(ses).Should(gexec.Exit(0))
		})
	})

	Describe("which", func() {
		DescribeTable("uml2ts",
			Entry(nil, "ts"),
			Entry(nil, "typescript"),
			Entry(nil, "uml2ts"),
			Entry(nil, "TypeScript"),
			Entry(nil, "tS"),
			func(ctx context.Context, input string) {
				expected := filepath.Join(gitRoot, "bin", "uml2ts")
				cmd := UxCommand(ctx, "which", input)

				ses, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)

				Expect(err).NotTo(HaveOccurred())
				Eventually(ses.Out).Should(gbytes.Say(expected + "\n"))
				Eventually(ses).Should(gexec.Exit(0))
			},
		)
	})
})
