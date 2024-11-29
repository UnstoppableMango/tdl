package internal_test

import (
	"bytes"
	"io"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/afero"

	"github.com/unstoppablemango/tdl/pkg/cmd/internal"
	"github.com/unstoppablemango/tdl/pkg/mediatype"
	"github.com/unstoppablemango/tdl/pkg/testing"
	. "github.com/unstoppablemango/tdl/pkg/testing/matcher"
)

var _ = Describe("Parse", func() {
	var (
		mockOs *testing.MockOs
	)

	BeforeEach(func() {
		mockOs = &testing.MockOs{}
	})

	Describe("ParseArgs", func() {
		It("should error when no arguments are provided", func() {
			_, err := internal.ParseArgs(mockOs, []string{})

			Expect(err).To(MatchError("no input file provided"))
		})

		It("should error when input file does not exist", func() {
			_, err := internal.ParseArgs(mockOs, []string{"some-file.txt"})

			Expect(err).To(MatchError("open some-file.txt: file does not exist"))
		})

		When("input is provided", func() {
			It("should error when file type can't be guessed", func() {
				_, err := mockOs.Fs().Create("input.txt")
				Expect(err).NotTo(HaveOccurred())

				_, err = internal.ParseArgs(mockOs, []string{"input.txt"})

				Expect(err).To(MatchError("unable to guess media type: input.txt"))
			})

			It("should write to stdout", func() {
				_, err := mockOs.Fs().Create("input.yaml")
				Expect(err).NotTo(HaveOccurred())
				buf := &bytes.Buffer{}
				mockOs.StdoutValue = buf

				res, err := internal.ParseArgs(mockOs, []string{"input.yaml"})

				Expect(err).NotTo(HaveOccurred())

				By("creating mock output")
				fs := afero.NewMemMapFs()
				_, err = fs.Create("thing.txt")
				Expect(err).NotTo(HaveOccurred())
				err = afero.WriteFile(fs, "thing.txt", []byte("testing"), os.ModePerm)
				Expect(err).NotTo(HaveOccurred())

				By("writing the mock output")
				Expect(res.Output.Write(fs)).To(Succeed())
				Expect(buf.String()).To(Equal("testing"))
			})
		})

		When("a dash is provided", func() {
			It("should read from stdin", func() {
				mockOs.StdinValue = testing.MockOsStdin(
					bytes.NewBufferString("testing"),
				)

				res, err := internal.ParseArgs(mockOs, []string{"-"})

				Expect(err).NotTo(HaveOccurred())
				Expect(res.Inputs).To(HaveLen(1))
				Expect(res.Inputs[0].MediaType()).To(Equal(mediatype.ApplicationProtobuf))
				data, err := io.ReadAll(res.Inputs[0])
				Expect(err).NotTo(HaveOccurred())
				Expect(string(data)).To(Equal("testing"))
			})

			When("stdin is empty", func() {
				It("should error", func() {
					mockOs.StdinValue = testing.MockTermStdin()

					_, err := internal.ParseArgs(mockOs, []string{"-"})

					Expect(err).To(MatchError("no input provided"))
				})
			})
		})

		When("input file exists", func() {
			BeforeEach(func() {
				_, err := mockOs.Fs().Create("input.yaml")
				Expect(err).NotTo(HaveOccurred())
			})

			It("should parse input file", func() {
				res, err := internal.ParseArgs(mockOs, []string{"input.yaml"})

				Expect(err).NotTo(HaveOccurred())
				Expect(res.Inputs).To(HaveLen(1))
				Expect(res.Inputs[0].MediaType()).To(Equal(mediatype.ApplicationYaml))
			})
		})

		When("input is a directory", func() {
			BeforeEach(func() {
				Expect(mockOs.Fs().Mkdir("test", os.ModeDir)).To(Succeed())
			})

			It("should error", func() {
				_, err := internal.ParseArgs(mockOs, []string{"test"})

				Expect(err).To(MatchError("test is a directory"))
			})
		})

		When("output is provided", func() {
			It("should write to output file", func() {
				_, err := mockOs.Fs().Create("input.yaml")
				Expect(err).NotTo(HaveOccurred())

				res, err := internal.ParseArgs(mockOs, []string{"input.yaml", "output.txt"})

				Expect(err).NotTo(HaveOccurred())

				By("creating mock output")
				fs := afero.NewMemMapFs()
				_, err = fs.Create("thing.txt")
				Expect(err).NotTo(HaveOccurred())
				err = afero.WriteFile(fs, "thing.txt", []byte("testing"), os.ModePerm)
				Expect(err).NotTo(HaveOccurred())

				By("writing the mock output")
				Expect(res.Output.Write(fs)).To(Succeed())
				Expect(mockOs.Fs()).To(ContainFile("output.txt"))
			})

			It("should write to existing output file", func() {
				_, err := mockOs.Fs().Create("input.yaml")
				Expect(err).NotTo(HaveOccurred())
				_, err = mockOs.Fs().Create("output.txt")
				Expect(err).NotTo(HaveOccurred())

				res, err := internal.ParseArgs(mockOs, []string{"input.yaml", "output.txt"})

				Expect(err).NotTo(HaveOccurred())

				By("creating mock output")
				fs := afero.NewMemMapFs()
				_, err = fs.Create("thing.txt")
				Expect(err).NotTo(HaveOccurred())
				err = afero.WriteFile(fs, "thing.txt", []byte("testing"), os.ModePerm)
				Expect(err).NotTo(HaveOccurred())

				By("writing the mock output")
				Expect(res.Output.Write(fs)).To(Succeed())
				Expect(mockOs.Fs()).To(ContainFile("output.txt"))
			})

			It("should write to output directory", func() {
				_, err := mockOs.Fs().Create("input.yaml")
				Expect(err).NotTo(HaveOccurred())
				Expect(mockOs.Fs().Mkdir("output", os.ModeDir)).To(Succeed())

				res, err := internal.ParseArgs(mockOs, []string{"input.yaml", "output"})

				Expect(err).NotTo(HaveOccurred())

				By("creating mock output")
				fs := afero.NewMemMapFs()
				_, err = fs.Create("thing.txt")
				Expect(err).NotTo(HaveOccurred())
				err = afero.WriteFile(fs, "thing.txt", []byte("testing"), os.ModePerm)
				Expect(err).NotTo(HaveOccurred())

				By("writing the mock output")
				Expect(res.Output.Write(fs)).To(Succeed())
				Expect(mockOs.Fs()).To(ContainFile("output/thing.txt"))
			})
		})

		It("should error when too many arguments are provided", func() {
			_, err := internal.ParseArgs(mockOs, []string{
				"a", "b", "c",
			})

			Expect(err).To(MatchError(ContainSubstring("too many arguments")))
		})
	})
})
