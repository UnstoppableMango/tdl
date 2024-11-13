package testing_test

import (
	"bytes"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/unstoppablemango/tdl/pkg/testing"
)

var _ = Describe("Cache", func() {
	It("should work", func() {
		cache := testing.NewCache(nil)
		data := bytes.NewBufferString("tdkfjdkhgsdl")

		err := cache.WriteAll("test-bin", data)

		Expect(err).NotTo(HaveOccurred())
	})
})
