package testing_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/afero"
	"github.com/unstoppablemango/tdl/pkg/testing"
)

var _ = Describe("Discover", func() {
	BeforeEach(func() {
		Expect(testdata).NotTo(BeNil())
	})

	It("should work", func() {
		fsys := afero.FromIOFS{FS: testdata}
		tests, err := testing.Discover(fsys, "testdata/validroot")

		Expect(err).NotTo(HaveOccurred())
		Expect(tests).To(HaveLen(2))
	})
})
