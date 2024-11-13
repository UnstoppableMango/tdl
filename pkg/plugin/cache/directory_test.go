package cache_test

import (
	"bytes"
	"io"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/unstoppablemango/tdl/pkg/plugin/cache"
)

var _ = Describe("Directory", func() {
	var root string

	BeforeEach(func() {
		tmp, err := os.MkdirTemp("", "")
		Expect(err).NotTo(HaveOccurred())
		root = tmp
	})

	AfterEach(func() {
		Expect(os.RemoveAll(root)).To(Succeed())
	})

	It("should cache a file", func() {
		cache := cache.AtDirectory(root)
		buf := bytes.NewBufferString("testing test")

		err := cache.WriteAll("thing", buf)

		Expect(err).NotTo(HaveOccurred())
		Expect(filepath.Join(root, "thing")).To(BeARegularFile())
	})

	It("should create the given path", func() {
		expected := filepath.Join(root, "subdir")
		cache := cache.AtDirectory(expected)
		buf := bytes.NewBufferString("fjdkslfkd")

		err := cache.WriteAll("thing", buf)

		Expect(err).NotTo(HaveOccurred())
		Expect(expected).To(BeADirectory())
	})

	Describe("when content is cached", func() {
		var (
			name     string
			contents []byte
			stub     cache.Directory
		)

		BeforeEach(func() {
			name = "test-file"
			contents = []byte("dfkdljsfkld")
			stub = cache.AtDirectory(root)
			buf := bytes.NewBuffer(contents)

			Expect(stub.WriteAll(name, buf)).To(Succeed())
		})

		It("should read the contents", func() {
			reader, err := stub.Reader(name)

			Expect(err).NotTo(HaveOccurred())
			data, err := io.ReadAll(reader)
			Expect(err).NotTo(HaveOccurred())
			Expect(data).To(Equal(contents))
		})

		It("should list the files", func() {
			files, err := stub.List()

			Expect(err).NotTo(HaveOccurred())
			Expect(files).To(ConsistOf(name))
		})
	})
})
