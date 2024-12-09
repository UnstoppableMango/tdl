package cache_test

import (
	"bytes"
	"fmt"
	"io"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	tdl "github.com/unstoppablemango/tdl/pkg"
	"github.com/unstoppablemango/tdl/pkg/cache"
)

var _ = Describe("Cache", func() {
	var mock tdl.Cache

	BeforeEach(func() {
		mock = cache.NewMemFs()
	})

	Describe("ErrNotExist", func() {
		It("should work the way I think go errors work", func() {
			match := cache.IsNotExist(cache.ErrNotExist)

			Expect(match).To(BeTrueBecause("I understand go errors"))
		})

		It("should work the way I think go error wrapping works", func() {
			match := cache.IsNotExist(fmt.Errorf("%w: %s", cache.ErrNotExist, "thing"))

			Expect(match).To(BeTrueBecause("I understand go error wrapping"))
		})
	})

	Describe("WriteString", func() {
		It("should write to the given cache item", func() {
			err := cache.WriteString(mock, "test", "blah")

			Expect(err).NotTo(HaveOccurred())
			item, err := mock.Get("test")
			Expect(err).NotTo(HaveOccurred())
			Expect(item).NotTo(BeNil())
			Expect(item.Name).To(Equal("test"))
			data, err := io.ReadAll(item)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(data)).To(Equal("blah"))
		})
	})

	Describe("WriteAll", func() {
		It("should write to the given cache item", func() {
			buf := bytes.NewBufferString("blah")
			err := cache.WriteAll(mock, "test", buf)

			Expect(err).NotTo(HaveOccurred())
			item, err := mock.Get("test")
			Expect(err).NotTo(HaveOccurred())
			Expect(item).NotTo(BeNil())
			Expect(item.Name).To(Equal("test"))
			data, err := io.ReadAll(item)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(data)).To(Equal("blah"))
		})
	})

	Describe("GetOrCreate", func() {
		It("should call create when the key does not exist", func() {
			sentinel := false

			item, err := cache.GetOrCreate(mock, "test",
				func() (io.ReadCloser, error) {
					sentinel = true
					return nil, nil
				},
			)

			Expect(err).NotTo(HaveOccurred())
			Expect(sentinel).To(BeTrueBecause("create was called"))
			Expect(item).NotTo(BeNil())
			Expect(item.Name).To(Equal("test"))
		})

		It("should not call create when the key exists", func() {
			sentinel := false
			err := cache.WriteString(mock, "test", "blah")
			Expect(err).NotTo(HaveOccurred())

			item, err := cache.GetOrCreate(mock, "test",
				func() (io.ReadCloser, error) {
					sentinel = true
					return nil, nil
				},
			)

			Expect(err).NotTo(HaveOccurred())
			Expect(sentinel).To(BeFalseBecause("create was not called"))
			Expect(item).NotTo(BeNil())
			Expect(item.Name).To(Equal("test"))
		})
	})
})
