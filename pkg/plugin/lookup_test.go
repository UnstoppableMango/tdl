package plugin

import (
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("LookupPath", func() {
	AfterEach(func() {
		_ = os.Unsetenv("BIN_DIR")
	})

	It("should search BIN_DIR", func() {
		err := os.Setenv("BIN_DIR", "expected")
		Expect(err).NotTo(HaveOccurred())

		_, err = LookupPath("na")
		Expect(err).To(MatchError(os.IsNotExist, "IsNotExist"))
	})

	It("should continue of BIN_DIR is unset", func() {
		_, err := LookupPath("na")
		Expect(err).To(MatchError("unable to find plugin: na"))
	})
})
