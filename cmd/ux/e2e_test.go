package main_test

import (
	"context"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/unstoppablemango/tdl/pkg/conform"
	"github.com/unstoppablemango/tdl/pkg/gen/cli"
)

var _ = Describe("End to end", func() {
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

			out, err := cmd.CombinedOutput()

			Expect(err).To(HaveOccurred())
			Expect(string(out)).To(Equal("no input file provided"))
		})

		It("should read spec from yaml file", FlakeAttempts(5), func(ctx context.Context) {
			input := filepath.Join(tsSuitePath(), "interface", "source.yml")
			output, err := os.ReadFile(filepath.Join(tsSuitePath(), "interface", "target.ts"))
			Expect(err).NotTo(HaveOccurred())
			cmd := UxCommand(ctx, "gen", "ts", input)

			out, err := cmd.CombinedOutput()

			Expect(err).NotTo(HaveOccurred(), string(out))
			Expect(string(out)).To(Equal(string(output)))
		})

		It("should error when input does not exist", func(ctx context.Context) {
			input := filepath.Join("fkjdslfkdjlsf")
			cmd := UxCommand(ctx, "gen", "ts", input)

			out, err := cmd.CombinedOutput()

			Expect(err).To(HaveOccurred())
			Expect(string(out)).To(Equal("parsing run config: stat fkjdslfkdjlsf: no such file or directory\n"))
		})

		It("should write to output file", FlakeAttempts(5), func(ctx context.Context) {
			input := filepath.Join(tsSuitePath(), "interface", "source.yml")
			tmp, err := os.MkdirTemp("", "")
			Expect(err).NotTo(HaveOccurred())
			output := filepath.Join(tmp, "index.ts")
			expected, err := os.ReadFile(filepath.Join(tsSuitePath(), "interface", "target.ts"))
			Expect(err).NotTo(HaveOccurred())
			cmd := UxCommand(ctx, "gen", "ts", input, output)

			out, err := cmd.CombinedOutput()

			Expect(err).NotTo(HaveOccurred(), string(out))
			Expect(string(out)).To(Equal(""))
			result, err := os.ReadFile(output)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(result)).To(Equal(string(expected)))
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

				out, err := cmd.CombinedOutput()

				Expect(err).NotTo(HaveOccurred())
				Expect(string(out)).To(Equal(expected + "\n"))
			},
		)
	})
})
