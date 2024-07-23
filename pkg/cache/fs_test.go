package cache_test

import (
	"bytes"
	"io"
	"log/slog"
	"os"
	"path"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/unstoppablemango/tdl/pkg/cache"
)

var _ = Describe("fsCache", func() {
	var workDir string
	var c cache.Cache

	BeforeEach(func() {
		By("Creating a temporary directory")
		tmp, err := os.MkdirTemp("", "fs_test")
		Expect(err).NotTo(HaveOccurred())

		workDir = tmp

		By("creating a new fsCache")
		c = cache.NewFsCache(workDir, slog.Default())
	})

	AfterEach(func() {
		By("removing the working directory")
		err := os.RemoveAll(workDir)
		Expect(err).NotTo(HaveOccurred())
	})

	It("should cache data", func() {
		data := bytes.NewBufferString("some data")

		err := c.Add("should-cache-data.txt", data)

		Expect(err).NotTo(HaveOccurred())
	})

	It("should retrieve cached data", func() {
		expectedText := "some data as well"
		fileName := "should-retrieve-cached-data.txt"

		By("writing a temporary file")
		err := os.WriteFile(
			path.Join(workDir, fileName),
			bytes.NewBufferString(expectedText).Bytes(),
			0600,
		)
		Expect(err).NotTo(HaveOccurred())

		reader, err := c.Get(fileName)

		Expect(err).NotTo(HaveOccurred())
		data, err := io.ReadAll(reader)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(data)).To(Equal(expectedText))
	})
})
