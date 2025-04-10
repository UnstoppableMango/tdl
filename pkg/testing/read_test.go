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
		test, err := testing.ReadRawTest(fsys, "testdata/validroot/validtest")

		Expect(err).NotTo(HaveOccurred())
		Expect(test).NotTo(BeNil())
		Expect(test.Name).To(Equal("validtest"))
		Expect(test.Input).NotTo(BeNil())
		Expect(test.Output).NotTo(BeNil())
	})

	It("should read source and target inputs", func() {
		fsys := afero.FromIOFS{FS: testdata}
		test, err := testing.ReadRawTest(fsys, "testdata/validroot/source_target")

		Expect(err).NotTo(HaveOccurred())
		Expect(test).NotTo(BeNil())
		Expect(test.Name).To(Equal("source_target"))
		Expect(test.Input).NotTo(BeNil())
		Expect(test.Output).NotTo(BeNil())
	})
})
