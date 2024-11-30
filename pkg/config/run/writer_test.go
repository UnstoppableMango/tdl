package run_test

import (
	"bytes"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/afero"
	"github.com/unstoppablemango/tdl/pkg/config/run"
)

var _ = Describe("Writer", func() {
	Describe("WriterOutput", func() {
		It("should write a file", func() {
			buf := &bytes.Buffer{}
			data := afero.NewMemMapFs()
			err := afero.WriteFile(data, "output.txt", []byte("testing"), os.ModePerm)
			Expect(err).NotTo(HaveOccurred())
			output := run.WriterOutput(buf)

			err = output.Write(data)

			Expect(err).NotTo(HaveOccurred())
			Expect(buf.String()).To(Equal("testing"))
		})

		It("should write a file in a directory", func() {
			buf := &bytes.Buffer{}
			data := afero.NewMemMapFs()
			Expect(data.Mkdir("blah", os.ModeDir)).To(Succeed())
			err := afero.WriteFile(data,
				"blah/output.txt",
				[]byte("testing"),
				os.ModePerm,
			)
			Expect(err).NotTo(HaveOccurred())
			output := run.WriterOutput(buf)

			err = output.Write(data)

			Expect(err).NotTo(HaveOccurred())
			Expect(buf.String()).To(Equal("testing"))
		})

		It("should write multiple files", func() {
			buf := &bytes.Buffer{}
			data := afero.NewMemMapFs()
			err := afero.WriteFile(data,
				"output1.txt",
				[]byte("testing1"),
				os.ModePerm,
			)
			Expect(err).NotTo(HaveOccurred())
			err = afero.WriteFile(data,
				"output2.txt",
				[]byte("testing2"),
				os.ModePerm,
			)
			Expect(err).NotTo(HaveOccurred())
			output := run.WriterOutput(buf)

			err = output.Write(data)

			Expect(err).NotTo(HaveOccurred())
			Expect(buf.String()).To(Equal("testing1testing2"))
		})
	})
})
