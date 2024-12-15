package cache_test

import (
	"bytes"
	"io"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/unstoppablemango/tdl/pkg/cache"
)

var _ = Describe("Tee", func() {
	It("should create a new reader", func() {
		c := cache.NewMemFs()

		r, err := cache.Tee(c, "", nil)

		Expect(err).NotTo(HaveOccurred())
		Expect(r).NotTo(BeNil())
	})

	It("should tee", func() {
		expected := []byte("bleh")
		c := cache.NewMemFs()
		b := bytes.NewReader(expected)

		r, err := cache.Tee(c, "test", b)

		Expect(err).NotTo(HaveOccurred())
		data, err := io.ReadAll(r)
		Expect(err).NotTo(HaveOccurred())
		Expect(data).To(Equal(expected))
		r, err = c.Get("test")
		Expect(err).NotTo(HaveOccurred())
		data, err = io.ReadAll(r)
		Expect(err).NotTo(HaveOccurred())
		Expect(data).To(Equal(expected))
	})
})
