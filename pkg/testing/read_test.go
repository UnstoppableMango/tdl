package testing_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/afero"

	"github.com/unstoppablemango/tdl/pkg/testing"
)

var _ = Describe("Read", func() {
	It("should work", func() {
		fsys := afero.FromIOFS{FS: testdata}
		test, err := testing.ReadTest(fsys, "testdata/validroot/validtest")

		Expect(err).NotTo(HaveOccurred())
		Expect(test).NotTo(BeNil())
		Expect(test.Name).To(Equal("validtest"))
		Expect(test.Input).NotTo(BeNil())
		Expect(test.Output).NotTo(BeNil())
	})
})
