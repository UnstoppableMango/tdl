package output_test

import (
	"bytes"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/afero"

	"github.com/unstoppablemango/tdl/pkg/gen/output"
)

var _ = Describe("Fs", func() {
	var fsys afero.Fs

	BeforeEach(func() {
		fsys = afero.NewMemMapFs()
	})

	It("should create a new file", func() {
		actual, err := output.Fs(fsys, "thing.txt")

		Expect(err).NotTo(HaveOccurred())
		Expect(actual).NotTo(BeNil())
		_, err = fsys.Stat("thing.txt")
		Expect(err).NotTo(HaveOccurred())
	})

	It("should create a new sink at directory", func() {
		err := fsys.Mkdir("dir", os.ModeDir)
		Expect(err).NotTo(HaveOccurred())

		actual, err := output.Fs(fsys, "dir")

		Expect(err).NotTo(HaveOccurred())
		err = actual.WriteUnit("thing.txt", bytes.NewBufferString("fkdjsfk"))
		Expect(err).NotTo(HaveOccurred())
		_, err = fsys.Stat("dir/thing.txt")
		Expect(err).NotTo(HaveOccurred())
	})

	Describe("ParseArgs", func() {
		It("should work", func() {
			err := fsys.Mkdir("dir", os.ModeDir)
			Expect(err).NotTo(HaveOccurred())

			actual, err := output.ParseArgs(fsys, []string{"not important", "dir"})

			Expect(err).NotTo(HaveOccurred())
			err = actual.WriteUnit("thing.txt", bytes.NewBufferString("fkdjsfk"))
			Expect(err).NotTo(HaveOccurred())
			_, err = fsys.Stat("dir/thing.txt")
			Expect(err).NotTo(HaveOccurred())
		})

		It("should not support three args", func() {
			_, err := output.ParseArgs(fsys, []string{"fdh", "fdj", "fdkjs"})

			Expect(err).To(HaveOccurred())
		})
	})
})
