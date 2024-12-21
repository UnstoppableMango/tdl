package run_test

import (
	"bytes"
	"io"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/afero"
	. "github.com/unmango/go/testing/matcher"

	"github.com/unstoppablemango/tdl/pkg/config/run"
	"github.com/unstoppablemango/tdl/pkg/mediatype"
	"github.com/unstoppablemango/tdl/pkg/testing"
	uxv1alpha1 "github.com/unstoppablemango/tdl/pkg/unmango/dev/ux/v1alpha1"
)

var _ = Describe("Config", func() {
	Describe("ParseInputs", func() {
		var os *testing.MockOs

		BeforeEach(func() {
			os = &testing.MockOs{}
		})

		It("should return empty when no inputs exist", func() {
			inputs, err := run.Inputs(os, &uxv1alpha1.RunConfig{
				Inputs: []*uxv1alpha1.Input{},
			})

			Expect(err).NotTo(HaveOccurred())
			Expect(inputs).To(BeEmpty())
		})

		When("stdin is specified", func() {
			It("should error when stdin is empty", func() {
				os.StdinValue = testing.MockTermStdin()
				_, err := run.Inputs(os, &uxv1alpha1.RunConfig{
					Inputs: []*uxv1alpha1.Input{
						{Value: &uxv1alpha1.Input_Stdin{Stdin: true}},
					},
				})

				Expect(err).To(MatchError("parsing run config: nothing on stdin"))
			})

			It("should read from stdin", func() {
				expected := "testing"
				buf := bytes.NewBufferString(expected)
				os.StdinValue = testing.MockOsStdin(buf)
				inputs, err := run.Inputs(os, &uxv1alpha1.RunConfig{
					Inputs: []*uxv1alpha1.Input{
						{Value: &uxv1alpha1.Input_Stdin{Stdin: true}},
					},
				})

				Expect(err).NotTo(HaveOccurred())
				Expect(inputs).To(HaveLen(1))
				input := inputs[0]
				Expect(input.MediaType()).To(Equal(mediatype.ApplicationProtobuf))
				data, err := io.ReadAll(input)
				Expect(err).NotTo(HaveOccurred())
				Expect(string(data)).To(Equal(expected))
			})
		})

		When("a file input is specified", func() {
			It("should error when input file does not exist", func() {
				_, err := run.Inputs(os, &uxv1alpha1.RunConfig{
					Inputs: []*uxv1alpha1.Input{
						{Value: &uxv1alpha1.Input_File{
							File: &uxv1alpha1.FileInput{Path: "blah.txt"},
						}},
					},
				})

				Expect(err).To(MatchError("parsing run config: open blah.txt: file does not exist"))
			})

			Describe("and the input file exists", func() {
				It("should error when the mediatype is unsupported", func() {
					_, err := os.Fs().Create("blah.txt")
					Expect(err).NotTo(HaveOccurred())

					_, err = run.Inputs(os, &uxv1alpha1.RunConfig{
						Inputs: []*uxv1alpha1.Input{
							{Value: &uxv1alpha1.Input_File{
								File: &uxv1alpha1.FileInput{Path: "blah.txt"},
							}},
						},
					})

					Expect(err).To(MatchError("parsing run config: unable to guess media type: blah.txt"))
				})

				It("should read the input file", func() {
					expected := "testing"
					err := afero.WriteFile(os.Fs(), "blah.yaml", []byte(expected), 0o777)
					Expect(err).NotTo(HaveOccurred())

					inputs, err := run.Inputs(os, &uxv1alpha1.RunConfig{
						Inputs: []*uxv1alpha1.Input{
							{Value: &uxv1alpha1.Input_File{
								File: &uxv1alpha1.FileInput{Path: "blah.yaml"},
							}},
						},
					})

					Expect(err).NotTo(HaveOccurred())
					Expect(inputs).To(HaveLen(1))
					input := inputs[0]
					Expect(input.MediaType()).To(Equal(mediatype.ApplicationYaml))
					data, err := io.ReadAll(input)
					Expect(err).NotTo(HaveOccurred())
					Expect(string(data)).To(Equal(expected))
				})
			})
		})
	})

	Describe("ParseOutput", func() {
		var os *testing.MockOs

		BeforeEach(func() {
			os = &testing.MockOs{}
		})

		It("should create stdout output", func() {
			buf := &bytes.Buffer{}
			os.StdoutValue = buf
			output, err := run.Output(os, &uxv1alpha1.RunConfig{
				Output: &uxv1alpha1.RunConfig_Stdout{Stdout: true},
			})

			Expect(err).NotTo(HaveOccurred())
			fs := afero.NewMemMapFs()
			err = afero.WriteFile(fs, "blah.output", []byte("testing"), 0o777)
			Expect(err).NotTo(HaveOccurred())
			err = output.Write(fs)
			Expect(err).NotTo(HaveOccurred())
			Expect(buf.String()).To(Equal("testing"))
		})

		It("should create fs output", func() {
			output, err := run.Output(os, &uxv1alpha1.RunConfig{
				Output: &uxv1alpha1.RunConfig_Path{
					Path: "blah.output",
				},
			})

			Expect(err).NotTo(HaveOccurred())
			fs := afero.NewMemMapFs()
			err = afero.WriteFile(fs, "blah.output", []byte("testing"), 0o777)
			Expect(err).NotTo(HaveOccurred())
			err = output.Write(fs)
			Expect(err).NotTo(HaveOccurred())
			Expect(os.Fs()).To(ContainFileWithBytes("blah.output", []byte("testing")))
		})
	})
})
