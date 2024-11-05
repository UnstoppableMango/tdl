package testing_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/unstoppablemango/tdl/pkg/testing"
)

var _ = Describe("Cache", func() {
	It("should work", func() {
		cache := testing.NewCache(nil)
		data := []byte("tdkfjdkhgsdl")

		err := cache.Write("test-bin", data)

		Expect(err).NotTo(HaveOccurred())
	})

	Describe("CacheForT", func() {
		It("should work", func() {
			data := []byte("dkfjslkdfjksdlf")

			err := testCacheForT.Write("test-bin", data)

			Expect(err).NotTo(HaveOccurred())
		})
	})
})
