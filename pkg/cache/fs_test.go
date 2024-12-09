package cache_test

import (
	"io"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/afero"
	. "github.com/unmango/go/testing/matcher"

	"github.com/unstoppablemango/tdl/pkg/cache"
)

var _ = Describe("Fs", func() {
	Describe("FsAt", func() {
		It("should error when the path is a file", func() {
			fs := afero.NewMemMapFs()
			err := afero.WriteFile(fs, "test", []byte("blah"), os.ModePerm)
			Expect(err).NotTo(HaveOccurred())

			_, err = cache.NewFsAt(fs, "test")

			Expect(err).To(MatchError(
				ContainSubstring("cache must be a directory: test")),
			)
		})

		It("should create the directory when it does not exist", func() {
			fs := afero.NewMemMapFs()

			_, err := cache.NewFsAt(fs, "test")

			Expect(err).NotTo(HaveOccurred())
			stat, err := fs.Stat("test")
			Expect(err).NotTo(HaveOccurred())
			Expect(stat.IsDir()).To(BeTrueBecause("the directory was created"))
		})
	})

	Describe("TmpFs", func() {
		It("should error when the key does not exist", func() {
			sut, err := cache.NewTmpFs()
			Expect(err).NotTo(HaveOccurred())

			_, err = sut.Get("test")

			Expect(err).To(MatchError("cache key does not exist: test"))
		})

		Describe("Happy Path", Ordered, func() {
			var sut *cache.Fs

			BeforeAll(func() {
				var err error
				sut, err = cache.NewTmpFs()
				Expect(err).NotTo(HaveOccurred())
			})

			It("should write to the given key", func() {
				writer, err := sut.Writer("test")

				Expect(err).NotTo(HaveOccurred())
				Expect(writer).NotTo(BeNil())
				_, err = io.WriteString(writer, "blah")
				Expect(err).NotTo(HaveOccurred())
				Expect(sut).To(ContainFileWithBytes("test", []byte("blah")))
			})

			It("should read the given key", func() {
				item, err := sut.Get("test")

				Expect(err).NotTo(HaveOccurred())
				Expect(item.Name).To(Equal("test"))
				data, err := io.ReadAll(item)
				Expect(err).NotTo(HaveOccurred())
				Expect(string(data)).To(Equal("blah"))
			})
		})
	})
})
