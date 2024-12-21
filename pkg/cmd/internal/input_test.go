package internal_test

import (
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/afero"
	. "github.com/unmango/go/testing/matcher"

	"github.com/unstoppablemango/tdl/pkg/cmd/internal"
)

var _ = Describe("Input", func() {
	Describe("FilterInput", func() {
		It("should match exact files", func() {
			fs := afero.NewMemMapFs()
			err := afero.WriteFile(fs, "test.txt", []byte("testing"), os.ModePerm)
			Expect(err).NotTo(HaveOccurred())

			r := internal.FilterInput(fs, "", []string{"test.txt"})

			Expect(r).To(ContainFile("test.txt"))
		})

		It("should match relative files", func() {
			fs := afero.NewMemMapFs()
			err := afero.WriteFile(fs, "test.txt", []byte("testing"), os.ModePerm)
			Expect(err).NotTo(HaveOccurred())

			r := internal.FilterInput(fs, "", []string{"./test.txt"})

			Expect(r).To(ContainFile("test.txt"))
		})

		It("should match absolute files", func() {
			fs := afero.NewMemMapFs()
			err := afero.WriteFile(fs, "/test.txt", []byte("testing"), os.ModePerm)
			Expect(err).NotTo(HaveOccurred())

			r := internal.FilterInput(fs, "/", []string{"/test.txt"})

			Expect(r).To(ContainFile("/test.txt"))
		})

		It("should match globs", func() {
			fs := afero.NewMemMapFs()
			err := fs.MkdirAll("/some/path", os.ModePerm)
			Expect(err).NotTo(HaveOccurred())
			err = afero.WriteFile(fs, "/some/path/test.txt", []byte("testing"), os.ModePerm)
			Expect(err).NotTo(HaveOccurred())

			r := internal.FilterInput(fs, "/", []string{"/some/*/test.txt"})

			Expect(r).To(ContainFile("/some/path/test.txt"))
		})

		It("should match multiple patterns", func() {
			fs := afero.NewMemMapFs()
			err := afero.WriteFile(fs, "test.txt", []byte("testing"), os.ModePerm)
			Expect(err).NotTo(HaveOccurred())

			r := internal.FilterInput(fs, "", []string{
				"test.yaml",
				"test.txt",
			})

			Expect(r).To(ContainFile("test.txt"))
		})

		It("should match regex", func() {
			fs := afero.NewMemMapFs()
			err := afero.WriteFile(fs, "test.txt", []byte("testing"), os.ModePerm)
			Expect(err).NotTo(HaveOccurred())

			r := internal.FilterInput(fs, "", []string{"test.(yaml|txt)"})

			Expect(r).To(ContainFile("test.txt"))
		})

		It("should ignore unmatched files", func() {
			fs := afero.NewMemMapFs()
			err := afero.WriteFile(fs, "test.txt", []byte("testing"), os.ModePerm)
			Expect(err).NotTo(HaveOccurred())

			r := internal.FilterInput(fs, "", []string{"test.yaml"})

			Expect(r).NotTo(ContainFile("test.txt"))
		})
	})
})
