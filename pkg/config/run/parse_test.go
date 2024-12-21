package run_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/unstoppablemango/tdl/pkg/config/run"
)

var _ = Describe("Parse", func() {
	Describe("ParseArgs", func() {
		It("should error when no arguments are provided", func() {
			_, err := run.ParseArgs([]string{})

			Expect(err).To(MatchError("not enough arguments"))
		})

		It("should error when input is not provided", func() {
			_, err := run.ParseArgs([]string{"ts"})

			Expect(err).To(MatchError("no input specified"))
		})

		It("should parse target", func() {
			config, err := run.ParseArgs([]string{"ts", "-"})

			Expect(err).NotTo(HaveOccurred())
			Expect(config.Target).To(Equal("ts"))
		})

		It("should parse stdin", func() {
			config, err := run.ParseArgs([]string{"ts", "-"})

			Expect(err).NotTo(HaveOccurred())
			Expect(config.Inputs).To(HaveLen(1))
			input := config.Inputs[0]
			Expect(input.GetStdin()).To(BeTrue())
			Expect(input.GetFile()).To(BeNil())
			Expect(input.GetInline()).To(BeNil())
		})

		It("should parse input file", func() {
			config, err := run.ParseArgs([]string{"ts", "some-file.txt"})

			Expect(err).NotTo(HaveOccurred())
			Expect(config.Inputs).To(HaveLen(1))
			input := config.Inputs[0]
			Expect(input.GetFile()).NotTo(BeNil())
			Expect(input.GetFile().Path).To(Equal("some-file.txt"))
			Expect(input.GetStdin()).To(BeFalse())
			Expect(input.GetInline()).To(BeNil())
		})

		It("should parse stdout when no output is provided", func() {
			config, err := run.ParseArgs([]string{"ts", "some-file.txt"})

			Expect(err).NotTo(HaveOccurred())
			Expect(config.Output).NotTo(BeNil())
			Expect(config.GetStdout()).To(BeTrue())
		})

		It("should parse output file", func() {
			config, err := run.ParseArgs([]string{"ts", "some-file.txt", "output.txt"})

			Expect(err).NotTo(HaveOccurred())
			Expect(config.Output).NotTo(BeNil())
			Expect(config.GetPath()).To(Equal("output.txt"))
			Expect(config.GetStdout()).To(BeFalse())
		})
	})
})
