package e2e_test

import (
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/unstoppablemango/tdl/pkg/testing/e2e"
)

var _ = Describe("Suite", func() {
	Describe("ReadSuite", func() {
		It("should work", func() {
			path := filepath.Join("testdata", "list")

			suite, err := e2e.ReadSuite(testfs, path)

			Expect(err).NotTo(HaveOccurred())
			Expect(suite.Name()).To(Equal("list"))
			var count int
			for _ = range suite.Tests() {
				count++
			}
			Expect(count).To(Equal(2))
		})
	})
})
