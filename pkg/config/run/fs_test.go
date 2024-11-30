package run_test

import (
	"io"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/afero"

	"github.com/unstoppablemango/tdl/pkg/config/run"
	"github.com/unstoppablemango/tdl/pkg/mediatype"
	"github.com/unstoppablemango/tdl/pkg/spec"
	. "github.com/unstoppablemango/tdl/pkg/testing/matcher"
	tdlv1alpha1 "github.com/unstoppablemango/tdl/pkg/unmango/dev/tdl/v1alpha1"
)

var _ = Describe("Fs", func() {
	Describe("OpenFile", func() {
		It("should error when file does not exist", func() {
			fs := afero.NewMemMapFs()

			_, err := run.OpenFile(fs, "blah")

			Expect(err).To(MatchError("open blah: file does not exist"))
		})

		It("should error when path is a directory", func() {
			fs := afero.NewMemMapFs()
			Expect(fs.Mkdir("blah", os.ModeDir)).To(Succeed())

			_, err := run.OpenFile(fs, "blah")

			Expect(err).To(MatchError("blah is a directory"))
		})

		It("should error when the media type is not supported", func() {
			fs := afero.NewMemMapFs()
			_, err := fs.Create("blah.txt")
			Expect(err).NotTo(HaveOccurred())

			_, err = run.OpenFile(fs, "blah.txt")

			Expect(err).To(MatchError("unable to guess media type: blah.txt"))
		})

		It("should open a yaml file", func() {
			fs := afero.NewMemMapFs()
			_, err := fs.Create("blah.yaml")
			Expect(err).NotTo(HaveOccurred())

			input, err := run.OpenFile(fs, "blah.yaml")

			Expect(err).NotTo(HaveOccurred())
			Expect(input.MediaType()).To(Equal(mediatype.ApplicationYaml))
		})

		It("should read the file content", func() {
			fs := afero.NewMemMapFs()
			data, err := spec.ToYaml(&tdlv1alpha1.Spec{
				Name: "Testing",
			})
			Expect(err).NotTo(HaveOccurred())
			err = afero.WriteFile(fs, "blah.yaml", data, os.ModePerm)
			Expect(err).NotTo(HaveOccurred())

			input, err := run.OpenFile(fs, "blah.yaml")

			Expect(err).NotTo(HaveOccurred())
			actual, err := io.ReadAll(input)
			Expect(err).NotTo(HaveOccurred())
			Expect(actual).To(Equal(data))
		})
	})

	Describe("FsOutput", func() {
		var (
			data    afero.Fs
			path    = "blah.output"
			content = "testing"
		)

		BeforeEach(func() {
			data = afero.NewMemMapFs()
			err := afero.WriteFile(data, path, []byte(content), os.ModePerm)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should write to a file", func() {
			fs := afero.NewMemMapFs()
			output := run.FsOutput(fs, "blah.output")

			err := output.Write(data)

			Expect(err).NotTo(HaveOccurred())
			Expect(fs).To(ContainFileWithBytes("blah.output", []byte(content)))
		})
	})
})
